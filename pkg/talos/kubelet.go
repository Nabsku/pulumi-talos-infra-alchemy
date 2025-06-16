package talos

import (
	"sigs.k8s.io/yaml"
)

// KubeletExtraArgs represents extra arguments for the kubelet.
type KubeletExtraArgs struct {
	Machine struct {
		Kubelet struct {
			ExtraArgs map[string]string `yaml:"extraArgs"`
		} `yaml:"kubelet"`
	} `yaml:"machine"`
}

// KubeletExtraConfig represents extra configuration for the kubelet.
type KubeletExtraConfig struct {
	Machine struct {
		Kubelet struct {
			ExtraConfig map[string]string `yaml:"extraConfig"`
		} `yaml:"kubelet"`
	} `yaml:"machine"`
}

// NewKubeletExtraArgsPatch creates a YAML patch for kubelet extra arguments.
func NewKubeletExtraArgsPatch(args map[string]string) ([]byte, error) {
	patch := KubeletExtraArgs{}
	patch.Machine.Kubelet.ExtraArgs = args
	return yaml.Marshal(patch)
}

// NewKubeletExtraConfigPatch creates a YAML patch for kubelet extra configuration.
func NewKubeletExtraConfigPatch(config map[string]string) ([]byte, error) {
	patch := KubeletExtraConfig{}
	patch.Machine.Kubelet.ExtraConfig = config
	return yaml.Marshal(patch)
}
