package types

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type NodeType int

const (
	ControlPlane NodeType = iota
	Worker
	Infrastructure
	Other // For any other node types that may be added in the future
)

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

type Node interface {
	Type() NodeType
	Pool() string
	SetPool(pool string)
	Name() string
	IP() pulumi.StringOutput
	SetName(name string)
	SetIP(ip pulumi.StringOutput)
	IsBootstrap() bool
	SetBootstrap(isBootstrap bool)
	Config() map[string]any
	SetVM(vm pulumi.Resource)
}
