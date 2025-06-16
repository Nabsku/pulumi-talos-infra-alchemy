package talos

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
)

// YamlToJSON converts YAML bytes to JSON bytes.
func YamlToJSON(yamlBytes []byte) ([]byte, error) {
	var data any
	if err := yaml.Unmarshal(yamlBytes, &data); err != nil {
		return nil, err
	}
	// Convert map[interface{}]interface{} to map[string]interface{}
	data = convertKeysToString(data)
	return json.Marshal(data)
}

// convertKeysToString recursively converts map keys to strings.
func convertKeysToString(i any) any {
	switch x := i.(type) {
	case map[any]any:
		m2 := map[string]any{}
		for k, v := range x {
			m2[fmt.Sprint(k)] = convertKeysToString(v)
		}
		return m2
	case []any:
		for i, v := range x {
			x[i] = convertKeysToString(v)
		}
	}
	return i
}
