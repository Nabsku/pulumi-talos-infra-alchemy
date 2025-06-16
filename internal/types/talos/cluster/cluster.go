package cluster

import (
	"errors"
	"fmt"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/client"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/cluster"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/machine"
	"proxmox-talos/internal/types"
	"proxmox-talos/internal/types/talos/nodes"
)

type Cluster struct {
	Name              string                         `json:"name"`
	Nodes             []types.Node                   `json:"nodes"`
	HasBootstrapNode  bool                           `json:"hasBootstrapNode"`
	TalosVersion      string                         `json:"talosVersion"`
	KubernetesVersion string                         `json:"kubernetesVersion"`
	KubernetesAPI     string                         `json:"kubernetesAPI"`
	MachineSecrets    *machine.Secrets               `json:"machineSecrets,omitempty"`
	ClientConfig      *client.GetConfigurationResult `json:"clientConfig,omitempty"`
}

func NewCluster(name, talosVersion, kubernetesVersion, kubernetesAPI string) *Cluster {
	return &Cluster{
		Name:              name,
		TalosVersion:      talosVersion,
		KubernetesVersion: kubernetesVersion,
		KubernetesAPI:     kubernetesAPI,
	}
}

func (c *Cluster) GenerateNodes(ctx *pulumi.Context, amount int, nodeType types.NodeType) error {
	var nodesList []types.Node
	for i := 0; i < amount; i++ {
		var node types.Node
		switch nodeType {
		case types.ControlPlane:
			node = &nodes.ControlPlaneNode{}
			if !c.HasBootstrapNode {
				node.SetBootstrap(true)
				c.HasBootstrapNode = true
			}
		case types.Worker:
			node = &nodes.WorkerNode{}
		default:
			return errors.New("unknown node type")
		}

		node.SetName(fmt.Sprintf("%s-%s-%d", c.Name, nodeType.String(), i))
		node.SetPool("default")
		nodesList = append(nodesList, node)
	}
	c.Nodes = append(c.Nodes, nodesList...)
	return nil
}

func (c *Cluster) GetNodesByType(nodeType types.NodeType) []string {
	nodesOfType := []string{}
	for _, node := range c.Nodes {
		if node.Type() == nodeType {
			nodesOfType = append(nodesOfType, node.Name())
		}
	}
	return nodesOfType
}

func (c *Cluster) GenerateMachineSecrets(ctx *pulumi.Context) error {
	machineSecrets, err := machine.NewSecrets(ctx, "talos-secrets", &machine.SecretsArgs{
		TalosVersion: pulumi.String(c.TalosVersion),
	})

	if err != nil {
		if err := ctx.Log.Error("Creating Talos Secrets failed with: "+err.Error(), nil); err != nil {
			return err
		}
	}
	c.MachineSecrets = machineSecrets
	return nil
}

func (c *Cluster) WaitForReady(ctx *pulumi.Context) {
	if len(c.Nodes) == 0 || c.ClientConfig == nil {
		return
	}

	args := &cluster.GetHealthArgs{
		ClientConfiguration: cluster.GetHealthClientConfiguration{
			ClientCertificate: c.ClientConfig.ClientConfiguration.ClientCertificate,
			ClientKey:         c.ClientConfig.ClientConfiguration.ClientKey,
			CaCertificate:     c.ClientConfig.ClientConfiguration.CaCertificate,
		},
		ControlPlaneNodes:    c.GetNodesByType(types.ControlPlane),
		Endpoints:            []string{c.KubernetesAPI},
		SkipKubernetesChecks: nil,
		Timeouts:             nil,
		WorkerNodes:          c.GetNodesByType(types.Worker),
	}

	_, err := cluster.GetHealth(ctx, args)
	if err != nil {
		if err := ctx.Log.Error("Waiting for Talos Cluster to be ready failed with: "+err.Error(), nil); err != nil {
			return
		}
	}

}

func (c *Cluster) String() string {
	return fmt.Sprintf("Cluster{Name: %s, Nodes: %d, HasBootstrapNode: %t, TalosVersion: %s, KubernetesVersion: %s, KubernetesAPI: %s}",
		c.Name, len(c.Nodes), c.HasBootstrapNode, c.TalosVersion, c.KubernetesVersion, c.KubernetesAPI)
}
