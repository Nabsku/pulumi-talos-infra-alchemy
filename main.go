package main

import (
	"fmt"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v6/go/proxmoxve"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v6/go/proxmoxve/cluster"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v6/go/proxmoxve/download"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v6/go/proxmoxve/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/imagefactory"
)

var (
	vms           = 3
	memory        = 8096
	cores         = 4
	diskSize      = 100
	network       = "vmbr1"
	talosArch     = "amd64"
	talosPlatform = "metal"
	talosVersion  = "v1.9.5"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		conf := config.New(ctx, "")

		proxmoxUsername := conf.Require("PROXMOX_USERNAME")
		password := conf.RequireSecret("PROXMOX_PASSWORD")
		proxmoxHost := conf.Require("PROXMOX_HOST")
		sideroLinkUrl := conf.Require("SIDEROLINK_URL")
		eventSink := conf.Require("EVENT_SINK")
		loggingKernel := conf.Require("LOGGING_KERNEL")

		proxmoxProvider, err := proxmoxve.NewProvider(ctx, "proxmoxve", &proxmoxve.ProviderArgs{
			Endpoint: pulumi.String(proxmoxHost),
			Username: pulumi.String(proxmoxUsername),
			Password: password,
			Insecure: pulumi.Bool(true),
		})
		if err != nil {
			err := ctx.Log.Error("Creating Proxmox Provider failed with: "+err.Error(), nil)
			if err != nil {
				return err
			}
		}

		talosImage, err := imagefactory.NewSchematic(ctx, "talos-image-from-factory", &imagefactory.SchematicArgs{
			Schematic: pulumi.String(`
customization:
  extraKernelArgs:
    - siderolink.api=` + sideroLinkUrl + `
    - talos.events.sink=` + eventSink + `
    - talos.logging.kernel=` + loggingKernel + `
  systemExtensions:
    officialExtensions:
      - siderolabs/amdgpu
      - siderolabs/amd-ucode
      - siderolabs/stargz-snapshotter
      - siderolabs/util-linux-tools
`)})
		if err != nil {
			err := ctx.Log.Error("Creating Talos Provider failed with: "+err.Error(), nil)
			if err != nil {
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
		if err != nil {
			err := ctx.Log.Error("Creating Talos Image failed with: "+err.Error(), nil)
			if err != nil {
				return err
			}
		}

		availableNodes, err := cluster.GetNodes(ctx, pulumi.Provider(proxmoxProvider))
		if err != nil {
			return err
		}

		downloadedImage, err := download.NewFile(ctx, "talos-image", &download.FileArgs{
			Url:         factoryOutput.Urls().Iso(),
			ContentType: pulumi.String("iso"),
			DatastoreId: pulumi.String("local"),
			NodeName:    pulumi.String(availableNodes.Names[0]),
		}, pulumi.Provider(proxmoxProvider))

		if err != nil {
			err := ctx.Log.Error("Downloading Talos Image failed with: "+err.Error(), nil)
			if err != nil {
				return err
			}
		}

		for i := range vms {
			vmName := fmt.Sprintf("talos-%d", i)

			_, err := vm.NewVirtualMachine(ctx, vmName, &vm.VirtualMachineArgs{
				NodeName: pulumi.String(availableNodes.Names[i%len(availableNodes.Names)]),
				Machine:  pulumi.String("q35"),
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
					},
				},
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
			}, pulumi.Provider(proxmoxProvider))
			if err != nil {
				return err
			}
		}
		return nil
	})
}
