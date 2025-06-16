package nodes

import (
	"fmt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"proxmox-talos/internal/types"
)

// WorkerNode represents a worker node in the cluster.
type WorkerNode struct {
	name     string
	ip       pulumi.StringOutput
	nodePool string
	vm       pulumi.Resource
}

// SetBootstrap is a no-op for worker nodes.
func (w *WorkerNode) SetBootstrap(isBootstrap bool) {
	_ = fmt.Sprintf("%s", isBootstrap)
}

// IsBootstrap always returns false for worker nodes.
func (w *WorkerNode) IsBootstrap() bool {
	return false
}

// SetVM sets the VM resource for this node.
func (w *WorkerNode) SetVM(vm pulumi.Resource) {
	w.vm = vm
}

// SetPool sets the node pool for this node.
func (w *WorkerNode) SetPool(pool string) {
	w.nodePool = pool
}

// Pool returns the node pool for this node.
func (w *WorkerNode) Pool() string {
	return w.nodePool
}

// Create is a stub for VM creation logic.
func (w *WorkerNode) Create() error {
	panic("Create not implemented for WorkerNode")
}

// Destroy is a stub for VM destruction logic.
func (w *WorkerNode) Destroy() error {
	panic("Destroy not implemented for WorkerNode")
}

// Name returns the name of the node.
func (w *WorkerNode) Name() string {
	return w.name
}

// SetName sets the name of the node.
func (w *WorkerNode) SetName(name string) {
	w.name = name
}

// IP returns the IP address of the node.
func (w *WorkerNode) IP() pulumi.StringOutput {
	return w.ip
}

// SetIP sets the IP address of the node.
func (w *WorkerNode) SetIP(ip pulumi.StringOutput) {
	w.ip = ip
}

// Type returns the node type.
func (w *WorkerNode) Type() types.NodeType {
	return types.Worker
}

// Config returns a map representation of the node's configuration.
func (w *WorkerNode) Config() map[string]any {
	return map[string]any{
		"name":     w.name,
		"ip":       w.ip,
		"nodePool": w.nodePool,
		"nodeType": w.Type().String(),
	}
}
