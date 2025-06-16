package proxmox

type VirtualMachine interface {
	Name() string
	IP() string
	Create() error
	Destroy() error
}
