package proxmox

import (
	"fmt"

	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve"
)

// Proxmox holds the provider and node information for a Proxmox environment.
type Proxmox struct {
	Provider     *proxmoxve.Provider
	ComputeNodes *[]ComputeNode
	Nodes        *[]VirtualMachine
}

// Validate checks if the Proxmox struct is properly configured.
func (p *Proxmox) Validate() error {
	if p.Provider == nil {
		return fmt.Errorf("Proxmox provider is not set")
	}
	if p.ComputeNodes == nil {
		return fmt.Errorf("Proxmox ComputeNodes is not set")
	}
	for i, node := range *p.ComputeNodes {
		if err := node.Validate(); err != nil {
			return fmt.Errorf("invalid compute node at index %d: %w", i, err)
		}
	}
	return nil
}
