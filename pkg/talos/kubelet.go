package talos

import (
	"sigs.k8s.io/yaml"
)

type KubeletExtraArgs struct {
	Machine struct {
		Kubelet struct {
			ExtraArgs map[string]string `yaml:"extraArgs"`
		} `yaml:"kubelet"`
	} `yaml:"machine"`
}

type KubeletExtraConfig struct {
	Machine struct {
		Kubelet struct {
			ExtraConfig map[string]string `yaml:"extraConfig"`
		} `yaml:"kubelet"`
	} `yaml:"machine"`
}

func NewKubeletExtraArgsPatch(args map[string]string) ([]byte, error) {
	patch := KubeletExtraArgs{}
	patch.Machine.Kubelet.ExtraArgs = args
	return yaml.Marshal(patch)
}

func NewKubeletExtraConfigPatch(config map[string]string) ([]byte, error) {
	patch := KubeletExtraConfig{}
	patch.Machine.Kubelet.ExtraConfig = config
	return yaml.Marshal(patch)
}
