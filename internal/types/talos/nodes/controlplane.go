package nodes

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"proxmox-talos/internal/types"
)

// ControlPlaneNode represents a control plane node in the cluster.
type ControlPlaneNode struct {
	name        string
	isBootstrap bool
	ip          pulumi.StringOutput
	nodePool    string
	vm          pulumi.Resource
}

// SetBootstrap sets whether this node is the bootstrap node.
func (c *ControlPlaneNode) SetBootstrap(isBootstrap bool) {
	c.isBootstrap = isBootstrap
}

// IsBootstrap returns true if this node is the bootstrap node.
func (c *ControlPlaneNode) IsBootstrap() bool {
	return c.isBootstrap
}

// VM returns the VM resource associated with this node.
func (c *ControlPlaneNode) VM() pulumi.Resource {
	return c.vm
}

// SetVM sets the VM resource for this node.
func (c *ControlPlaneNode) SetVM(vm pulumi.Resource) {
	c.vm = vm
}

// SetPool sets the node pool for this node.
func (c *ControlPlaneNode) SetPool(pool string) {
	c.nodePool = pool
}

// Pool returns the node pool for this node.
func (c *ControlPlaneNode) Pool() string {
	return c.nodePool
}

// Create is a stub for VM creation logic.
func (c *ControlPlaneNode) Create() error {
	panic("Create not implemented for ControlPlaneNode")
}

// Destroy is a stub for VM destruction logic.
func (c *ControlPlaneNode) Destroy() error {
	panic("Destroy not implemented for ControlPlaneNode")
}

// Name returns the name of the node.
func (c *ControlPlaneNode) Name() string {
	return c.name
}

// SetName sets the name of the node.
func (c *ControlPlaneNode) SetName(name string) {
	c.name = name
}

// IP returns the IP address of the node.
func (c *ControlPlaneNode) IP() pulumi.StringOutput {
	return c.ip
}

// SetIP sets the IP address of the node.
func (c *ControlPlaneNode) SetIP(ip pulumi.StringOutput) {
	c.ip = ip
}

// Config returns a map representation of the node's configuration.
func (c *ControlPlaneNode) Config() map[string]any {
	return map[string]any{
		"name": c.name,
		"ip":   c.ip,
		"type": c.Type().String(),
	}
}

// Type returns the node type.
func (c *ControlPlaneNode) Type() types.NodeType {
	return types.ControlPlane
}
