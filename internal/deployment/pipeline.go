package deployment

import (
	"fmt"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve/cluster"
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve/download"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/pulumiverse/pulumi-talos/sdk/go/talos/imagefactory"
	internalConfig "proxmox-talos/internal/config"
	"proxmox-talos/internal/types"
	talosCluster "proxmox-talos/internal/types/talos/cluster"
	"proxmox-talos/pkg/proxmox"
	"proxmox-talos/pkg/talos"
	"strings"
)

// Pipeline represents the deployment pipeline
type Pipeline struct {
	ctx     *pulumi.Context
	config  *internalConfig.ClusterConfig
	cluster *talosCluster.Cluster
	proxmox *proxmox.Proxmox
}

// NewPipeline creates a new deployment pipeline
func NewPipeline(ctx *pulumi.Context, cfg *internalConfig.ClusterConfig) (*Pipeline, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	cluster := talosCluster.NewCluster(
		cfg.ClusterName,
		cfg.TalosVersion,
		cfg.KubernetesVersion,
		cfg.ApiVIP,
	)

	return &Pipeline{
		ctx:     ctx,
		config:  cfg,
		cluster: cluster,
	}, nil
}

// Execute runs the complete deployment pipeline
func (p *Pipeline) Execute() error {
	if err := p.setupCluster(); err != nil {
		return fmt.Errorf("cluster setup failed: %w", err)
	}

	if err := p.setupProxmox(); err != nil {
		return fmt.Errorf("proxmox setup failed: %w", err)
	}

	if err := p.createInfrastructure(); err != nil {
		return fmt.Errorf("infrastructure creation failed: %w", err)
	}

	if err := p.deployTalos(); err != nil {
		return fmt.Errorf("talos deployment failed: %w", err)
	}

	if err := p.generateOutputs(); err != nil {
		return fmt.Errorf("output generation failed: %w", err)
	}

	return nil
}

// setupCluster initializes the cluster configuration
func (p *Pipeline) setupCluster() error {
	if err := p.cluster.GenerateNodes(p.config.ControlPlaneCount, types.ControlPlane); err != nil {
		return fmt.Errorf("generating control plane nodes: %w", err)
	}

	if err := p.cluster.GenerateNodes(p.config.WorkerCount, types.Worker); err != nil {
		return fmt.Errorf("generating worker nodes: %w", err)
	}

	if err := p.cluster.GenerateMachineSecrets(p.ctx); err != nil {
		return fmt.Errorf("generating machine secrets: %w", err)
	}

	p.ctx.Log.Info(fmt.Sprintf("Generated Talos Cluster: %s", p.cluster.String()), nil)
	return nil
}

// setupProxmox initializes the Proxmox provider
func (p *Pipeline) setupProxmox() error {
	conf := config.New(p.ctx, "")
	var err error
	p.proxmox, err = proxmox.NewProxmox(p.ctx, conf)
	if err != nil {
		return fmt.Errorf("creating proxmox client: %w", err)
	}

	if err := p.proxmox.GatherHosts(p.ctx); err != nil {
		return fmt.Errorf("gathering proxmox hosts: %w", err)
	}

	return nil
}

// createInfrastructure creates VMs and downloads images
func (p *Pipeline) createInfrastructure() error {
	// Create Talos image
	talosImage, err := talos.CreateTalosImage(p.ctx, p.config)
	if err != nil {
		return fmt.Errorf("creating talos image: %w", err)
	}

	// Download image to Proxmox
	downloadedImage, err := p.proxmox.DownloadTalosImage(p.ctx, talosImage)
	if err != nil {
		return fmt.Errorf("downloading talos image: %w", err)
	}

	// Create VMs
	if err := p.createVMs(downloadedImage); err != nil {
		return fmt.Errorf("creating VMs: %w", err)
	}

	return nil
}

// createVMs creates all the virtual machines
func (p *Pipeline) createVMs(downloadedImage *download.File) error {
	availableNodes, err := p.proxmox.GetAvailableNodes(p.ctx)
	if err != nil {
		return fmt.Errorf("getting available nodes: %w", err)
	}

	for i, node := range p.cluster.Nodes {
		p.ctx.Log.Info(fmt.Sprintf("Creating VM for node: %s", node.Name()), nil)

		vmConfig := proxmox.VMConfig{
			Name:          node.Name(),
			NodeName:      availableNodes[i%len(availableNodes)],
			Cores:         p.config.Cores,
			MemoryMB:      p.config.Memory,
			DiskSizeGB:    p.config.DiskSize,
			NetworkBridge: p.config.Network,
			CdromFileID:   downloadedImage.ID(),
			Provider:      p.proxmox.Provider,
			DependsOn:     []pulumi.Resource{downloadedImage},
		}

		createdVM, ip, err := proxmox.CreateVM(p.ctx, vmConfig)
		if err != nil {
			return fmt.Errorf("creating VM for node %s: %w", node.Name(), err)
		}

		node.SetIP(ip)
		node.SetVM(createdVM)
	}

	return nil
}

// deployTalos configures and bootstraps the Talos cluster
func (p *Pipeline) deployTalos() error {
	deployer := talos.NewDeployer(p.ctx, p.cluster, p.config)
	return deployer.Deploy()
}

// generateOutputs creates the final outputs like kubeconfig
func (p *Pipeline) generateOutputs() error {
	if err := p.cluster.WaitForReady(p.ctx); err != nil {
		return fmt.Errorf("waiting for cluster ready: %w", err)
	}

	p.ctx.Log.Info(fmt.Sprintf("Cluster %s is ready", p.cluster.Name), nil)

	return talos.GenerateKubeconfig(p.ctx, p.cluster)
}
