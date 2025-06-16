package nodes

import (
	"fmt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"proxmox-talos/internal/types"
)

type WorkerNode struct {
	name     string
	ip       pulumi.StringOutput // Using pulumi.StringOutput for Pulumi integration
	nodePool string
}

func (w *WorkerNode) SetBootstrap(isBootstrap bool) {
	// Worker nodes do not have a bootstrap state, this method is a no-op
	_ = fmt.Sprintf("%s", isBootstrap)
	fmt.Println("Worker nodes do not have a bootstrap state.")
}

func (w *WorkerNode) IsBootstrap() bool {
	return false // Worker nodes are not bootstrap nodes
}

func (w *WorkerNode) SetVM(vm pulumi.Resource) {
	//TODO implement me
	panic("implement me")
}

func (w *WorkerNode) SetPool(pool string) {
	w.nodePool = pool
}

func (w *WorkerNode) Pool() string {
	return w.nodePool
}

func (w *WorkerNode) Create() error {
	//TODO implement me
	panic("implement me")
}

func (w *WorkerNode) Destroy() error {
	//TODO implement me
	panic("implement me")
}

func (w *WorkerNode) Name() string {
	return w.name
}

func (w *WorkerNode) SetName(name string) {
	w.name = name
}

func (w *WorkerNode) IP() pulumi.StringOutput {
	return w.ip
}

func (w *WorkerNode) SetIP(ip pulumi.StringOutput) {
	w.ip = ip
}

func (w *WorkerNode) Type() types.NodeType {
	return types.Worker
}

// Config returns a map representation of the WorkerNode configuration
// TODO: find a way to generate talos manifests.
func (w *WorkerNode) Config() map[string]any {
	return map[string]any{
		"name":     w.name,
		"ip":       w.ip,
		"nodePool": w.nodePool,
		"nodeType": w.Type().String(),
	}
}
