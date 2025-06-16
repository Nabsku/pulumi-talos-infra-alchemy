package proxmox

import (
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve/cluster"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"proxmox-talos/internal/types/proxmox"
)

// Proxmox is a wrapper around the internal Proxmox type for Pulumi integration.
type Proxmox proxmox.Proxmox
type VirtualMachine proxmox.VirtualMachine

// NewProxmox initializes a new Proxmox instance with the provided configuration.
func NewProxmox(ctx *pulumi.Context, conf *config.Config) (*Proxmox, error) {
	proxmoxUsername := conf.Require("PROXMOX_USERNAME")
	password := conf.RequireSecret("PROXMOX_PASSWORD")
	proxmoxHost := conf.Require("PROXMOX_HOST")

	proxmoxProvider, err := proxmoxve.NewProvider(ctx, "proxmoxve", &proxmoxve.ProviderArgs{
		Endpoint: pulumi.String(proxmoxHost),
		Username: pulumi.String(proxmoxUsername),
		Password: password,
		Insecure: pulumi.Bool(true),
	})

	if err != nil {
		return nil, ctx.Log.Error("Creating Proxmox Provider failed with: "+err.Error(), nil)
	}

	return &Proxmox{
		Provider: proxmoxProvider,
		Nodes:    &[]proxmox.VirtualMachine{},
	}, nil
}

// GatherHosts retrieves the available Proxmox nodes and populates the ComputeNodes slice.
func (p *Proxmox) GatherHosts(ctx *pulumi.Context) error {
	if p.ComputeNodes == nil {
		p.ComputeNodes = &[]proxmox.ComputeNode{}
	}
	// Gather available Proxmox nodes
	availableNodes, err := cluster.GetNodes(ctx, pulumi.Provider(p.Provider))
	if err != nil {
		return ctx.Log.Error("Gathering Proxmox hosts failed with: "+err.Error(), nil)
	}

	if len(availableNodes.Names) == 0 {
		ctx.Log.Error("No Proxmox nodes found", nil)
	}

	for i, node := range availableNodes.Names {
		if node == "" {
			ctx.Log.Error("Node name is empty at index "+string(rune(i)), nil)
		}

		if !availableNodes.Onlines[i] {
			ctx.Log.Debug("Node "+node+" is offline", nil)
			continue
		}

		ctx.Log.Info("Node "+node+" is online", nil)

		newNode := proxmox.ComputeNode{}
		newNode.SetName(node)
		newNode.SetCPU(availableNodes.CpuCounts[i])
		newNode.SetMemory(availableNodes.MemoryAvailables[i])

		*p.ComputeNodes = append(*p.ComputeNodes, newNode)
	}

	return nil
}
