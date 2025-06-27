package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"proxmox-talos/internal/file"
	"proxmox-talos/internal/types"
	talosCluster "proxmox-talos/internal/types/talos/cluster"
	"proxmox-talos/pkg/proxmox"
	"proxmox-talos/pkg/talos"

	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve/cluster"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	tsdk "github.com/pulumiverse/pulumi-talos/sdk/go/talos/cluster"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/imagefactory"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/machine"
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
	talosAPIVIP       = "https://192.168.4.9:6443"
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

		tCluster := talosCluster.NewCluster(talosClusterName, talosVersion, kubernetesVersion, talosAPIVIP)
		if err := tCluster.GenerateNodes(controlPlaneCount, types.ControlPlane); err != nil {
			return ctx.Log.Error("Generating control plane nodes failed with: "+err.Error(), nil)
		}
		if err := tCluster.GenerateNodes(workerCount, types.Worker); err != nil {
			return ctx.Log.Error("Generating worker nodes failed with: "+err.Error(), nil)
		}

		if err := tCluster.GenerateMachineSecrets(ctx); err != nil {
			return err
		}

		ctx.Log.Info(fmt.Sprintf("Generated Talos Cluster: %s", tCluster.String()), nil)
		ctx.Log.Info(fmt.Sprintf("Nodes: %v", tCluster.Nodes), nil)

		proxmoxStruct, err := proxmox.NewProxmox(ctx, conf)
		if err != nil {
			return err
		}

		if err = proxmoxStruct.GatherHosts(ctx); err != nil {
			return err
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
			return ctx.Log.Error("Creating Talos Provider failed with: "+err.Error(), nil)
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
			return ctx.Log.Error("Downloading Talos Image failed with: "+err.Error(), nil)
		}

		// Create VMs
		for i, node := range tCluster.Nodes {
			ctx.Log.Info(fmt.Sprintf("Creating VM for node: %s", node.Name()), nil)
			vmConfig := proxmox.VMConfig{
				Name:          node.Name(),
				NodeName:      availableNodes.Names[i%len(availableNodes.Names)],
				Cores:         cores,
				MemoryMB:      memory,
				DiskSizeGB:    diskSize,
				NetworkBridge: network,
				CdromFileID:   downloadedImage.ID(),
				Provider:      proxmoxStruct.Provider,
				DependsOn:     []pulumi.Resource{downloadedImage},
			}
			createdVM, ip, err := proxmox.CreateVM(ctx, vmConfig)
			if err != nil {
				return ctx.Log.Error(fmt.Sprintf("Failed to create VM for node %s: %v", node.Name(), err), nil)
			}
			node.SetIP(ip)
			node.SetVM(createdVM)
		}

		for _, node := range tCluster.Nodes {
			ctx.Log.Info(fmt.Sprintf("Creating Talos Node for type %s and name %s", node.Type().String(), node.Name()), nil)
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

			var mergedConfig string
			files, err := file.GatherPatchFilesInDir(fmt.Sprintf("talos-config/%v", node.Type().String()))
			if err != nil {
				return fmt.Errorf("failed to gather %v configuration files: %w", node.Type().String(), err)
			}
			fmt.Printf("Found %d configuration files for %s\n", len(files), node.Type().String())
			if len(files) == 0 {
				return fmt.Errorf("no control plane configuration files found")
			}
			mergedConfig, err = talos.MergeYaml(files...)
			if err != nil {
				return fmt.Errorf("failed to merge %v configuration files: %w", node.Type().String(), err)
			}

			tmpl, err := template.New("config").Parse(mergedConfig)
			if err != nil {
				return fmt.Errorf("failed to read %v API configuration: %w", node.Type().String(), err)
			}

			var rendered bytes.Buffer
			err = tmpl.Execute(&rendered, node)
			if err != nil {
				return fmt.Errorf("failed to render %v API configuration: %w", node.Type().String(), err)
			}

			cpPatch, err := talos.YamlToJSON(rendered.Bytes())
			if err != nil {
				return fmt.Errorf("failed to convert %v API configuration to JSON: %w", node.Type().String(), err)
			}
			cpPatchStr = string(cpPatch)

			configPatches := pulumi.StringArray{pulumi.String(json0)}
			if cpPatchStr != "" {
				configPatches = append(configPatches, pulumi.String(cpPatchStr))
			}

			node.IP().ApplyT(func(nodeIP string) error {
				applyArgs := machine.ConfigurationApplyArgs{
					ClientConfiguration:       tCluster.MachineSecrets.ClientConfiguration,
					MachineConfigurationInput: configuration.MachineConfiguration(),
					Node:                      pulumi.String(node.Name()),
					ConfigPatches:             configPatches,
					Endpoint:                  pulumi.String(nodeIP),
				}

				apply, err := machine.NewConfigurationApply(ctx, fmt.Sprintf("%s-configuration-apply", node.Name()),
					&applyArgs)
				if err != nil {
					return err
				}

				if node.IsBootstrap() {
					bootstrapArgs := machine.BootstrapArgs{
						ClientConfiguration: tCluster.MachineSecrets.ClientConfiguration,
						Node:                pulumi.String(nodeIP),
					}

					customTimeouts := pulumi.CustomTimeouts{
						Create: "10m",
					}

					bootstrap, err := machine.NewBootstrap(ctx, "bootstrap",
						&bootstrapArgs, pulumi.DependsOn([]pulumi.Resource{apply}),
						pulumi.Timeouts(&customTimeouts))

					kc := tCluster.MachineSecrets.ClientConfiguration.ApplyT(func(clientConfig machine.ClientConfiguration) (any, error) {
						return tCluster.Nodes[0].IP().ApplyT(func(nodeIP string) (any, error) {
							args := &tsdk.KubeconfigArgs{
								ClientConfiguration: &tsdk.KubeconfigClientConfigurationArgs{
									ClientCertificate: pulumi.String(clientConfig.ClientCertificate),
									ClientKey:         pulumi.String(clientConfig.ClientKey),
									CaCertificate:     pulumi.String(clientConfig.CaCertificate),
								},
								Node: pulumi.String(nodeIP),
							}

							k, err := tsdk.NewKubeconfig(ctx, "talos-kubeconfig", args, pulumi.DependsOn([]pulumi.Resource{bootstrap, tCluster.MachineSecrets}))
							if err != nil {
								return nil, fmt.Errorf("failed to generate kubeconfig: %w", err)
							}

							k.KubeconfigRaw.ApplyT(func(kubeconfig string) error {
								file.WriteToFile("kubeconfig.yaml", kubeconfig)
								return nil
							})

							return k, nil
						}), nil
					})

					tCluster.Kubeconfig = kc

					if err != nil {
						return err
					}
					ctx.Log.Info(fmt.Sprintf("Node %s is bootstrapped", node.Name()), nil)
				}
				return nil
			})
		}

		// Export the Talos cluster secrets
		ctx.Export("MachineSecrets", tCluster.MachineSecrets)
		ctx.Export("Kubeconfig", tCluster.Kubeconfig)
		ctx.Log.Info(fmt.Sprintf("Cluster %s is ready", tCluster.Name), nil)

		ctx.Log.Info("Pulumi Talos Proxmox deployment completed successfully", nil)

		return nil
	})
}
