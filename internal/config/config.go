package config

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// ClusterConfig holds all cluster configuration
type ClusterConfig struct {
	ControlPlaneCount int      `json:"controlPlaneCount"`
	WorkerCount       int      `json:"workerCount"`
	Memory            int      `json:"memory"`
	Cores             int      `json:"cores"`
	DiskSize          int      `json:"diskSize"`
	Network           string   `json:"network"`
	TalosArch         string   `json:"talosArch"`
	TalosPlatform     string   `json:"talosPlatform"`
	ClusterName       string   `json:"clusterName"`
	TalosVersion      string   `json:"talosVersion"`
	ApiVIP            string   `json:"apiVIP"`
	Extensions        []string `json:"extensions"`
	KubernetesVersion string   `json:"kubernetesVersion"`
}

// LoadConfig loads configuration from Pulumi config with sensible defaults
func LoadConfig(ctx *pulumi.Context) *ClusterConfig {
	conf := config.New(ctx, "")

	// Helper function to get int with default
	getIntOrDefault := func(key string, def int) int {
		if val := conf.GetInt(key); val != 0 {
			return val
		}
		return def
	}

	// Helper function to get string with default
	getStringOrDefault := func(key string, def string) string {
		if val := conf.Get(key); val != "" {
			return val
		}
		return def
	}

	return &ClusterConfig{
		ControlPlaneCount: getIntOrDefault("controlPlaneCount", 3),
		WorkerCount:       getIntOrDefault("workerCount", 3),
		Memory:            getIntOrDefault("memory", 8096),
		Cores:             getIntOrDefault("cores", 4),
		DiskSize:          getIntOrDefault("diskSize", 100),
		Network:           getStringOrDefault("network", "vmbr1"),
		TalosArch:         getStringOrDefault("talosArch", "amd64"),
		TalosPlatform:     getStringOrDefault("talosPlatform", "metal"),
		ClusterName:       getStringOrDefault("clusterName", "talos"),
		TalosVersion:      getStringOrDefault("talosVersion", "v1.10.0"),
		ApiVIP:            getStringOrDefault("apiVIP", "https://192.168.4.9:6443"),
		KubernetesVersion: getStringOrDefault("kubernetesVersion", "v1.33.0"),
		Extensions: []string{
			"siderolabs/amdgpu",
			"siderolabs/amd-ucode",
			"siderolabs/stargz-snapshotter",
			"siderolabs/util-linux-tools",
			"siderolabs/qemu-guest-agent",
		},
	}
}

// Validate checks if the configuration is valid
func (c *ClusterConfig) Validate() error {
	if c.ControlPlaneCount%2 == 0 {
		return fmt.Errorf("control plane count must be odd, got %d", c.ControlPlaneCount)
	}
	return nil
}
