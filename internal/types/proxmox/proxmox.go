package proxmox

import (
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve"
)

type Proxmox struct {
	Provider     *proxmoxve.Provider
	ComputeNodes *[]ComputeNode
	Nodes        *[]VirtualMachine
}
