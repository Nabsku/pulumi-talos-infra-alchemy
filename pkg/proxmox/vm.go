package proxmox

import (
	"fmt"

	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// We do this to fix a weird issue with the Proxmox provider where it always updates the VM disk speed settings
var magicNumber = 9999999

// VMConfig holds the configuration for creating a Proxmox VM.
type VMConfig struct {
	Name          string
	NodeName      string
	Cores         int
	MemoryMB      int
	DiskSizeGB    int
	NetworkBridge string
	CdromFileID   pulumi.IDOutput
	Provider      pulumi.ProviderResource
	DependsOn     []pulumi.Resource
}

// Validate checks the VMConfig for required fields and returns an error if any are missing or invalid.
func (cfg *VMConfig) Validate() error {
	if cfg.Name == "" {
		return fmt.Errorf("VMConfig: Name is required")
	}
	if cfg.NodeName == "" {
		return fmt.Errorf("VMConfig: NodeName is required")
	}
	if cfg.Cores <= 0 {
		return fmt.Errorf("VMConfig: Cores must be > 0")
	}
	if cfg.MemoryMB <= 0 {
		return fmt.Errorf("VMConfig: MemoryMB must be > 0")
	}
	if cfg.DiskSizeGB <= 0 {
		return fmt.Errorf("VMConfig: DiskSizeGB must be > 0")
	}
	if cfg.NetworkBridge == "" {
		return fmt.Errorf("VMConfig: NetworkBridge is required")
	}
	if cfg.Provider == nil {
		return fmt.Errorf("VMConfig: Provider is required")
	}
	return nil
}

// CreateVM creates a new Proxmox VM with the given configuration and returns the VM resource and its IP output.
func CreateVM(ctx *pulumi.Context, cfg VMConfig) (*vm.VirtualMachine, pulumi.StringOutput, error) {
	if err := cfg.Validate(); err != nil {
		return nil, pulumi.String("").ToStringOutput(), fmt.Errorf("invalid VMConfig: %w", err)
	}

	vmArgs := &vm.VirtualMachineArgs{
		NodeName: pulumi.String(cfg.NodeName),
		Name:     pulumi.String(cfg.Name),
		Agent: &vm.VirtualMachineAgentArgs{
			Enabled: pulumi.Bool(true),
		},
		Machine: pulumi.String("q35"),
		Cpu: &vm.VirtualMachineCpuArgs{
			Cores:   pulumi.Int(cfg.Cores),
			Sockets: pulumi.Int(1),
			Numa:    pulumi.Bool(true),
			Type:    pulumi.String("x86-64-v2-AES"),
		},
		Memory: &vm.VirtualMachineMemoryArgs{
			Dedicated: pulumi.Int(cfg.MemoryMB),
		},
		Disks: &vm.VirtualMachineDiskArray{
			&vm.VirtualMachineDiskArgs{
				Interface:   pulumi.String("virtio0"),
				Size:        pulumi.Int(cfg.DiskSizeGB),
				DatastoreId: pulumi.String("local"),
				Speed: &vm.VirtualMachineDiskSpeedArgs{
					IopsRead:           pulumi.Int(magicNumber),
					IopsReadBurstable:  pulumi.Int(magicNumber),
					IopsWrite:          pulumi.Int(magicNumber),
					IopsWriteBurstable: pulumi.Int(magicNumber),
					Read:               pulumi.Int(magicNumber),
					ReadBurstable:      pulumi.Int(magicNumber),
					Write:              pulumi.Int(magicNumber),
					WriteBurstable:     pulumi.Int(magicNumber),
				},
			},
		},
		BootOrders:    pulumi.StringArray{pulumi.String("virtio0"), pulumi.String("ide3")},
		StopOnDestroy: pulumi.Bool(true),
		OperatingSystem: &vm.VirtualMachineOperatingSystemArgs{
			Type: pulumi.String("l26"),
		},
		NetworkDevices: &vm.VirtualMachineNetworkDeviceArray{
			&vm.VirtualMachineNetworkDeviceArgs{
				Bridge: pulumi.String(cfg.NetworkBridge),
			},
		},
		Cdrom: &vm.VirtualMachineCdromArgs{
			FileId: cfg.CdromFileID,
		},
	}

	opts := []pulumi.ResourceOption{
		pulumi.Provider(cfg.Provider),
	}
	if len(cfg.DependsOn) > 0 {
		opts = append(opts, pulumi.DependsOn(cfg.DependsOn))
	}

	createdVM, err := vm.NewVirtualMachine(ctx, cfg.Name, vmArgs, opts...)
	if err != nil {
		return nil, pulumi.String("").ToStringOutput(), fmt.Errorf("failed to create VM %s: %w", cfg.Name, err)
	}

	ip := createdVM.Ipv4Addresses.ApplyT(func(ipv4 [][]string) string {
		if len(ipv4) == 0 {
			return ""
		}
		lastInner := ipv4[len(ipv4)-1]
		if len(lastInner) > 0 {
			return lastInner[len(lastInner)-1]
		}
		return ""
	}).(pulumi.StringOutput)

	return createdVM, ip, nil
}
