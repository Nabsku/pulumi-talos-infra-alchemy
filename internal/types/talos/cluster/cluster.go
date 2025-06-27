package cluster

import (
	"errors"
	"fmt"

	"proxmox-talos/internal/types"
	"proxmox-talos/internal/types/talos/nodes"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/client"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/cluster"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/machine"
)

// Cluster represents a Talos cluster and its configuration.
type Cluster struct {
	Name              string                         `json:"name"`
	Nodes             []types.Node                   `json:"nodes"`
	HasBootstrapNode  bool                           `json:"hasBootstrapNode"`
	TalosVersion      string                         `json:"talosVersion"`
	KubernetesVersion string                         `json:"kubernetesVersion"`
	KubernetesAPI     string                         `json:"kubernetesAPI"`
	MachineSecrets    *machine.Secrets               `json:"machineSecrets,omitempty"`
	ClientConfig      *client.GetConfigurationResult `json:"clientConfig,omitempty"`
	Kubeconfig        pulumi.Output                  `json:"kubeconfig,omitempty"`
}

// NewCluster creates a new Cluster instance.
func NewCluster(name, talosVersion, kubernetesVersion, kubernetesAPI string) *Cluster {
	return &Cluster{
		Name:              name,
		TalosVersion:      talosVersion,
		KubernetesVersion: kubernetesVersion,
		KubernetesAPI:     kubernetesAPI,
	}
}

// GenerateNodes creates the specified number of nodes of the given type and adds them to the cluster.
func (c *Cluster) GenerateNodes(amount int, nodeType types.NodeType) error {
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
		c.Nodes = append(c.Nodes, node)
	}
	return nil
}

// GetNodesByType returns the names of nodes of the specified type.
func (c *Cluster) GetNodesByType(nodeType types.NodeType) []pulumi.StringOutput {
	var nodesOfType []pulumi.StringOutput
	for _, node := range c.Nodes {
		if node.Type() == nodeType {
			nodesOfType = append(nodesOfType, node.IP())
		}
	}
	return nodesOfType
}

// GenerateMachineSecrets creates Talos machine secrets for the cluster.
func (c *Cluster) GenerateMachineSecrets(ctx *pulumi.Context) error {
	machineSecrets, err := machine.NewSecrets(ctx, "talos-secrets", &machine.SecretsArgs{
		TalosVersion: pulumi.String(c.TalosVersion),
	})
	if err != nil {
		if logErr := ctx.Log.Error("Creating Talos Secrets failed with: "+err.Error(), nil); logErr != nil {
			return logErr
		}
		return err
	}
	c.MachineSecrets = machineSecrets
	return nil
}

// WaitForReady waits for the Talos cluster to become healthy.
func (c *Cluster) WaitForReady(ctx *pulumi.Context) pulumi.Output {
	if len(c.Nodes) == 0 {
		_ = ctx.Log.Error("WaitForReady: cluster nodes are not set", nil)
	}

	controlPlaneIPs := c.GetNodesByType(types.ControlPlane)
	if len(controlPlaneIPs) == 0 {
		_ = ctx.Log.Error("WaitForReady: no control plane node IPs found", nil)
	}

	controlPlaneInputs := make([]interface{}, len(controlPlaneIPs))
	for i, ip := range controlPlaneIPs {
		controlPlaneInputs[i] = ip
	}

	// Combine node IPs and MachineSecrets.ClientConfiguration Output
	allOutputs := append(controlPlaneInputs, c.MachineSecrets.ClientConfiguration)
	return pulumi.All(allOutputs...).ApplyT(func(args []interface{}) (string, error) {
		var ipStrings []string
		for i := 0; i < len(controlPlaneInputs); i++ {
			if s, ok := args[i].(string); ok {
				ipStrings = append(ipStrings, s)
			}
		}
		clientConfig, ok := args[len(args)-1].(*machine.ClientConfiguration)
		if !ok || clientConfig == nil {
			_ = ctx.Log.Error("WaitForReady: MachineSecrets.ClientConfiguration is not set", nil)
			return "MachineSecrets.ClientConfiguration is not set", errors.New("MachineSecrets.ClientConfiguration is not set")
		}
		healthArgs := &cluster.GetHealthArgs{
			ClientConfiguration: cluster.GetHealthClientConfiguration{
				ClientCertificate: clientConfig.ClientCertificate,
				ClientKey:         clientConfig.ClientKey,
				CaCertificate:     clientConfig.CaCertificate,
			},
			ControlPlaneNodes:    ipStrings,
			Endpoints:            []string{c.KubernetesAPI},
			SkipKubernetesChecks: nil,
		}
		_, err := cluster.GetHealth(ctx, healthArgs)
		if err != nil {
			_ = ctx.Log.Error("WaitForReady: GetHealth failed: "+err.Error(), nil)
			return "GetHealth failed", err
		}
		return "ok", nil
	})
}

func (c *Cluster) GenerateKubeconfig(ctx *pulumi.Context) error {
	if c.ClientConfig == nil {
		return errors.New("client configuration is not set")
	}

	args := &cluster.KubeconfigArgs{
		ClientConfiguration: &cluster.KubeconfigClientConfigurationArgs{
			ClientCertificate: c.MachineSecrets.ClientConfiguration.ClientCertificate(),
			ClientKey:         c.MachineSecrets.ClientConfiguration.ClientKey(),
			CaCertificate:     c.MachineSecrets.ClientConfiguration.CaCertificate(),
		},
		Node: c.Nodes[0].IP(),
	}

	fmt.Println(args)

	k, err := cluster.NewKubeconfig(ctx, "talos-kubeconfig", args, pulumi.DependsOn([]pulumi.Resource{c.MachineSecrets}))
	if err != nil {
		return fmt.Errorf("failed to generate kubeconfig: %w", err)
	}
	ctx.Export("kubeconfig", k.KubeconfigRaw)
	return nil
}

// String returns a string representation of the cluster.
func (c *Cluster) String() string {
	return fmt.Sprintf("Cluster{Name: %s, Nodes: %d, HasBootstrapNode: %t, TalosVersion: %s, KubernetesVersion: %s, KubernetesAPI: %s}",
		c.Name, len(c.Nodes), c.HasBootstrapNode, c.TalosVersion, c.KubernetesVersion, c.KubernetesAPI)
}
