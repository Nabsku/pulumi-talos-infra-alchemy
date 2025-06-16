package talos

import (
	"encoding/json"
	"sigs.k8s.io/yaml"
)

type Config struct {
	Machine struct {
		Network struct {
			Interfaces []struct {
				DeviceSelector struct {
					Driver string `yaml:"driver" json:"driver"`
				} `yaml:"deviceSelector" json:"deviceSelector"`
				Vip struct {
					IP string `yaml:"ip" json:"ip"`
				} `yaml:"vip" json:"vip"`
			} `yaml:"interfaces" json:"interfaces"`
		} `yaml:"network" json:"network"`
	} `yaml:"machine" json:"machine"`
}

func (c *Config) Marshal(b []byte) []byte {
	var config Config
	if err := yaml.Unmarshal(b, &config); err != nil {
		return []byte("")
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		return []byte("")
	}
	return jsonData

}
