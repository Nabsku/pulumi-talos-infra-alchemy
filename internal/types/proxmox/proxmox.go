package proxmox

import (
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve"
)

// Proxmox holds the provider and node information for a Proxmox environment.
type Proxmox struct {
	Provider     *proxmoxve.Provider
	ComputeNodes *[]ComputeNode
	Nodes        *[]VirtualMachine
}
