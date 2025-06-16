package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve/cluster"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/imagefactory"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/machine"
	talosclient "github.com/pulumiverse/pulumi-talos/sdk/go/talos/client"
	taloscluster "github.com/pulumiverse/pulumi-talos/sdk/go/talos/cluster"
	"proxmox-talos/internal/types"
	talosCluster "proxmox-talos/internal/types/talos/cluster"
	"proxmox-talos/pkg/proxmox"
	patches "proxmox-talos/pkg/talos"
	"strings"
	"text/template"
)

var (
	controlPlaneCount = 3
	workerCount       = 3
	memory            = 8096
	cores             = 4
	diskSize          = 100
	network           = "vmbr1"
	talosArch         = "amd64"
	talosPlatform     = "metal"
	talosClusterName  = "talos"
	talosVersion      = "v1.10.0"
	talosApiVIP       = "https://192.168.4.9:6443"
	extensions        = []string{
		"siderolabs/amdgpu",
		"siderolabs/amd-ucode",
		"siderolabs/stargz-snapshotter",
		"siderolabs/util-linux-tools",
		"siderolabs/qemu-guest-agent",
	}
	kubernetesVersion = "v1.33.0"
)

func isOdd(n int) bool {
	return n%2 != 0
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		conf := config.New(ctx, "")

		if !isOdd(controlPlaneCount) {
			return fmt.Errorf("control plane count must be odd")
		}

		tCluster := talosCluster.NewCluster(talosClusterName, talosVersion, kubernetesVersion, talosApiVIP)
		if err := tCluster.GenerateNodes(ctx, controlPlaneCount, types.ControlPlane); err != nil {
			return ctx.Log.Error("Generating control plane nodes failed with: "+err.Error(), nil)
		}
		if err := tCluster.GenerateNodes(ctx, workerCount, types.Worker); err != nil {
			return ctx.Log.Error("Generating worker nodes failed with: "+err.Error(), nil)
		}

		err := tCluster.GenerateMachineSecrets(ctx)
		if err != nil {
			return err
		}

		fmt.Println("Generated Talos Cluster:", tCluster)
		fmt.Println("Nodes:", tCluster.Nodes)

		proxmoxStruct, err := proxmox.NewProxmox(ctx, conf)
		if err != nil {
			if err := ctx.Log.Error("Creating Proxmox Provider failed with: "+err.Error(), nil); err != nil {
				return err
			}
		}

		if err := proxmoxStruct.GatherHosts(ctx); err != nil {
			if err := ctx.Log.Error("Gathering Proxmox hosts failed with: "+err.Error(), nil); err != nil {
				return err
			}
		}

		talosImage, err := imagefactory.NewSchematic(ctx, "talos-image-from-factory", &imagefactory.SchematicArgs{
			Schematic: pulumi.String(fmt.Sprintf(`
customization:
  systemExtensions:
    officialExtensions:
      - %s
`, strings.Join(extensions, "\n      - "))),
		})
		if err != nil {
			if err := ctx.Log.Error("Creating Talos Provider failed with: "+err.Error(), nil); err != nil {
				return err
			}
		}

		factoryArgs := imagefactory.GetUrlsOutputArgs{
			Architecture: pulumi.String(talosArch),
			Platform:     pulumi.String(talosPlatform),
			SchematicId:  talosImage.ID().ToStringOutput(),
			TalosVersion: pulumi.String(talosVersion),
		}
		factoryOutput := imagefactory.GetUrlsOutput(ctx, factoryArgs, nil)

		availableNodes, err := cluster.GetNodes(ctx, pulumi.Provider(proxmoxStruct.Provider))
		if err != nil {
			return err
		}

		downloadedImage, err := proxmoxStruct.DownloadTalosImage(ctx, &factoryOutput)
		if err != nil {
			if err := ctx.Log.Error("Downloading Talos Image failed with: "+err.Error(), nil); err != nil {
				return err
			}
		}

		for i, node := range tCluster.Nodes {
			fmt.Printf("Creating VM for node: %s\n", node.Name())
			createdVM, err := vm.NewVirtualMachine(ctx, node.Name(), &vm.VirtualMachineArgs{
				NodeName: pulumi.String(availableNodes.Names[i%len(availableNodes.Names)]),
				Name:     pulumi.String(node.Name()),
				Agent: &vm.VirtualMachineAgentArgs{
					Enabled: pulumi.Bool(true),
				},
				Machine: pulumi.String("q35"),
				Cpu: &vm.VirtualMachineCpuArgs{
					Cores:   pulumi.Int(cores),
					Sockets: pulumi.Int(1),
					Numa:    pulumi.Bool(true),
					Type:    pulumi.String("x86-64-v2-AES"),
				},
				Memory: &vm.VirtualMachineMemoryArgs{
					Dedicated: pulumi.Int(memory),
				},
				Disks: &vm.VirtualMachineDiskArray{
					&vm.VirtualMachineDiskArgs{
						Interface:   pulumi.String("virtio0"),
						Size:        pulumi.Int(diskSize),
						DatastoreId: pulumi.String("local"),
						Speed: &vm.VirtualMachineDiskSpeedArgs{
							IopsRead:           pulumi.Int(0),
							IopsReadBurstable:  pulumi.Int(0),
							IopsWrite:          pulumi.Int(0),
							IopsWriteBurstable: pulumi.Int(0),
							Read:               pulumi.Int(0),
							ReadBurstable:      pulumi.Int(0),
							Write:              pulumi.Int(0),
							WriteBurstable:     pulumi.Int(0),
						},
					},
				},
				SerialDevices: nil,
				BootOrders:    pulumi.StringArray{pulumi.String("virtio0"), pulumi.String("ide3")},
				StopOnDestroy: pulumi.Bool(true),
				OperatingSystem: &vm.VirtualMachineOperatingSystemArgs{
					Type: pulumi.String("l26"),
				},
				NetworkDevices: &vm.VirtualMachineNetworkDeviceArray{
					&vm.VirtualMachineNetworkDeviceArgs{
						Bridge: pulumi.String(network),
					},
				},
				Cdrom: &vm.VirtualMachineCdromArgs{
					FileId: downloadedImage.ID(),
				},
			}, pulumi.Provider(proxmoxStruct.Provider), pulumi.DependsOn([]pulumi.Resource{downloadedImage}))
			if err != nil {
				return err
			}

			ip := createdVM.Ipv4Addresses.ApplyT(func(ipv4 [][]string) string {
				lastInner := ipv4[len(ipv4)-1]
				if len(lastInner) > 0 {
					return lastInner[len(lastInner)-1]
				}
				return ""
			}).(pulumi.StringOutput)

			node.SetIP(ip)
		}

		// --- BEGIN: Use Talos provider to generate talosconfig and kubeconfig ---
		// Gather all control plane node IPs as []pulumi.StringInput
		var controlPlaneIPs []pulumi.StringInput
		for _, node := range tCluster.Nodes {
			if node.Type() == types.ControlPlane {
				controlPlaneIPs = append(controlPlaneIPs, node.IP())
			}
		}

		// Generate Talos client config (talosconfig)
		talosConfig := talosclient.GetConfigurationOutput(ctx, &talosclient.GetConfigurationOutputArgs{
			ClusterName:     pulumi.String(tCluster.Name),
			ClusterEndpoint: pulumi.String(tCluster.KubernetesAPI),
			MachineSecrets:  tCluster.MachineSecrets.MachineSecrets,
		}, nil)

		// Generate kubeconfig using Talos provider
		kubeConfig := taloscluster.GetKubeconfigOutput(ctx, &taloscluster.GetKubeconfigOutputArgs{
			ClientConfiguration: talosConfig.ClientConfiguration(),
			Endpoints:           pulumi.ToStringArray(controlPlaneIPs),
		}, nil)

		ctx.Export("talosconfig", talosConfig.Config())
		ctx.Export("kubeconfig", kubeConfig.Kubeconfig())
		// --- END: Use Talos provider to generate talosconfig and kubeconfig ---

		for _, node := range tCluster.Nodes {
			fmt.Printf("Creating Talos Node for type %s and name %s\n", node.Type().String(), node.Name())
			configuration := machine.GetConfigurationOutput(ctx, machine.GetConfigurationOutputArgs{
				ClusterName:     pulumi.String(tCluster.Name),
				MachineType:     pulumi.String(node.Type().String()),
				ClusterEndpoint: pulumi.String(tCluster.KubernetesAPI),
				Docs:            pulumi.Bool(false),
				Examples:        pulumi.Bool(false),
				MachineSecrets:  tCluster.MachineSecrets.MachineSecrets,
			}, nil)

			tmpJSON0, err := json.Marshal(map[string]any{
				"machine": map[string]any{
					"install": map[string]any{
						"disk": "/dev/vda",
					},
				},
			})
			if err != nil {
				return err
			}

			json0 := string(tmpJSON0)
			var cpPatchStr string

			if node.Type() == types.ControlPlane {
				var rendered bytes.Buffer
				tmpl, err := template.ParseFiles("talos-config/controlplane/api.yaml.tmpl")
				if err != nil {
					return fmt.Errorf("failed to read controlplane API configuration: %w", err)
				}
				err = tmpl.Execute(&rendered, node)
				if err != nil {
					return fmt.Errorf("failed to render controlplane API configuration: %w", err)
				}

				cpPatch, err := patches.YamlToJSON(rendered.Bytes())
				if err != nil {
					return fmt.Errorf("failed to convert controlplane API configuration to JSON: %w", err)
				}
				cpPatchStr = string(cpPatch)
			}

			configPatches := pulumi.StringArray{pulumi.String(json0)}
			if cpPatchStr != "" {
				configPatches = append(configPatches, pulumi.String(cpPatchStr))
			}

			applyArgs := machine.ConfigurationApplyArgs{
				ClientConfiguration:       tCluster.MachineSecrets.ClientConfiguration,
				MachineConfigurationInput: configuration.MachineConfiguration(),
				Node:                      pulumi.String(node.Name()),
				ConfigPatches:             configPatches,
				Endpoint:                  node.IP(),
			}

			apply, err := machine.NewConfigurationApply(ctx, fmt.Sprintf("%s-configuration-apply", node.Name()),
				&applyArgs)
			if err != nil {
				if err := ctx.Log.Error("Creating Talos Configuration Apply failed with: "+err.Error(), nil); err != nil {
					return err
				}
			}

			if node.IsBootstrap() {
				bootstrapArgs := machine.BootstrapArgs{
					ClientConfiguration: tCluster.MachineSecrets.ClientConfiguration,
					Node:                node.IP(),
				}

				customTimeouts := pulumi.CustomTimeouts{
					Create: "10m",
				}

				_, err := machine.NewBootstrap(ctx, "bootstrap",
					&bootstrapArgs, pulumi.DependsOn([]pulumi.Resource{apply}),
					pulumi.Timeouts(&customTimeouts))

				if err != nil {
					if err := ctx.Log.Error("Creating Talos Bootstrap failed with: "+err.Error(), nil); err != nil {
						return err
					}
				} else {
					fmt.Printf("Node %s is bootstrapped\n", node.Name())
				}
			}

			tCluster.WaitForReady(ctx)
			fmt.Printf("Cluster %s is ready\n", tCluster.Name)

		}
		if err := ctx.Log.Info("Pulumi Talos Proxmox deployment completed successfully", nil); err != nil {
			ctx.Log.Error("Logging completion message failed: "+err.Error(), nil)
		}

		return nil
	})
}
