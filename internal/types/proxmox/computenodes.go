package proxmox

import "net"

// ComputeNode represents a physical or virtual compute node in Proxmox.
type ComputeNode struct {
	name   string
	ip     net.IP
	cpu    int
	memory int
}

// Name returns the name of the compute node.
func (c *ComputeNode) Name() string {
	return c.name
}

// SetName sets the name of the compute node.
func (c *ComputeNode) SetName(name string) {
	c.name = name
}

// IP returns the IP address of the compute node.
func (c *ComputeNode) IP() net.IP {
	return c.ip
}

// SetIP sets the IP address of the compute node.
func (c *ComputeNode) SetIP(ip net.IP) {
	c.ip = ip
}

// CPU returns the number of CPUs of the compute node.
func (c *ComputeNode) CPU() int {
	return c.cpu
}

// SetCPU sets the number of CPUs of the compute node.
func (c *ComputeNode) SetCPU(cpu int) {
	c.cpu = cpu
}

// Memory returns the memory (in MB) of the compute node.
func (c *ComputeNode) Memory() int {
	return c.memory
}

// SetMemory sets the memory (in MB) of the compute node.
func (c *ComputeNode) SetMemory(memory int) {
	c.memory = memory
}
