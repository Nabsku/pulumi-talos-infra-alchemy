package proxmox

// VirtualMachine defines the interface for VM operations in Proxmox.
type VirtualMachine interface {
	Name() string
	IP() string
	Create() error
	Destroy() error
}
