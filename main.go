package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve/cluster"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/imagefactory"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/machine"
	"proxmox-talos/internal/types"
	talosCluster "proxmox-talos/internal/types/talos/cluster"
	"proxmox-talos/pkg/proxmox"
	"proxmox-talos/pkg/talos"
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

		if err := tCluster.GenerateMachineSecrets(ctx); err != nil {
			return err
		}

		ctx.Log.Info(fmt.Sprintf("Generated Talos Cluster: %s", tCluster.String()), nil)
		ctx.Log.Info(fmt.Sprintf("Nodes: %v", tCluster.Nodes), nil)

		proxmoxStruct, err := proxmox.NewProxmox(ctx, conf)
		if err != nil {
			return err
		}

		if err := proxmoxStruct.GatherHosts(ctx); err != nil {
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

		// Use the new VMConfig and CreateVM abstraction for VM creation
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

				cpPatch, err := talos.YamlToJSON(rendered.Bytes())
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
				return ctx.Log.Error("Creating Talos Configuration Apply failed with: "+err.Error(), nil)
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
					return ctx.Log.Error("Creating Talos Bootstrap failed with: "+err.Error(), nil)
				}
				ctx.Log.Info(fmt.Sprintf("Node %s is bootstrapped", node.Name()), nil)
			}

			tCluster.WaitForReady(ctx)
			ctx.Log.Info(fmt.Sprintf("Cluster %s is ready", tCluster.Name), nil)
		}

		if err := ctx.Log.Info("Pulumi Talos Proxmox deployment completed successfully", nil); err != nil {
			ctx.Log.Error("Logging completion message failed: "+err.Error(), nil)
		}

		return nil
	})
}
