package types

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// NodeType represents the type of a node in the cluster.
type NodeType int

const (
	ControlPlane NodeType = iota
	Worker
	Infrastructure
	Other // For any other node types that may be added in the future
)

// String returns the string representation of the NodeType.
func (n NodeType) String() string {
	switch n {
	case ControlPlane:
		return "controlplane"
	case Worker:
		return "worker"
	case Infrastructure:
		return "infrastructure"
	case Other:
		return "other"
	default:
		return "unknown"
	}
}

// Node is the interface that all cluster nodes must implement.
type Node interface {
	Type() NodeType
	Pool() string
	SetPool(pool string)
	Name() string
	SetName(name string)
	IP() pulumi.StringOutput
	SetIP(ip pulumi.StringOutput)
	IsBootstrap() bool
	SetBootstrap(isBootstrap bool)
	Config() map[string]any
	SetVM(vm pulumi.Resource)
}
