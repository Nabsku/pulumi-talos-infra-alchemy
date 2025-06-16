package nodes

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"proxmox-talos/internal/types"
)

type ControlPlaneNode struct {
	name        string
	isBootstrap bool                // Indicates if this node is the bootstrap control plane node
	ip          pulumi.StringOutput // Using pulumi.StringOutput for Pulumi integration
	nodePool    string
	vm          pulumi.Resource // This will hold the VM resource for the control plane node
}

func (c *ControlPlaneNode) SetBootstrap(isBootstrap bool) {
	c.isBootstrap = isBootstrap
}

func (c *ControlPlaneNode) IsBootstrap() bool {
	return c.isBootstrap
}

func (c *ControlPlaneNode) VM() pulumi.Resource {
	return c.vm
}

func (c *ControlPlaneNode) SetVM(vm pulumi.Resource) {
	c.vm = vm
}

func (c *ControlPlaneNode) SetPool(pool string) {
	c.nodePool = pool
}

func (c *ControlPlaneNode) Pool() string {
	return c.nodePool
}

func (c *ControlPlaneNode) Create() error {
	//TODO implement me
	panic("implement me")
}

func (c *ControlPlaneNode) Destroy() error {
	//TODO implement me
	panic("implement me")
}

func (c *ControlPlaneNode) Name() string {
	return c.name
}

func (c *ControlPlaneNode) SetName(name string) {
	c.name = name
}

func (c *ControlPlaneNode) IP() pulumi.StringOutput {
	return c.ip
}

func (c *ControlPlaneNode) SetIP(ip pulumi.StringOutput) {
	c.ip = ip
}

// Config returns a map representation of the ControlPlaneNode configuration
// TODO: find a way to generate talos manifests.
func (c *ControlPlaneNode) Config() map[string]any {
	return map[string]any{
		"name": c.name,
		"ip":   c.ip,
		"type": c.Type().String(),
	}
}

func (c *ControlPlaneNode) Type() types.NodeType {
	return types.ControlPlane
}
