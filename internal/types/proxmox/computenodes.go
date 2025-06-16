package proxmox

import "net"

type ComputeNode struct {
	name   string
	ip     net.IP
	cpu    int
	memory int
}

func (c *ComputeNode) Name() string {
	return c.name
}

func (c *ComputeNode) SetName(name string) {
	c.name = name
}

func (c *ComputeNode) IP() net.IP {
	return c.ip
}

func (c *ComputeNode) SetIP(ip net.IP) {
	c.ip = ip
}

func (c *ComputeNode) CPU() int {
	return c.cpu
}

func (c *ComputeNode) SetCPU(cpu int) {
	c.cpu = cpu
}

func (c *ComputeNode) Memory() int {
	return c.memory
}

func (c *ComputeNode) SetMemory(memory int) {
	c.memory = memory
}
