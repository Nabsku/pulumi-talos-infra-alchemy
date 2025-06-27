package talos

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/TwiN/deepmerge"
	"gopkg.in/yaml.v3"
)

// YamlToJSON converts YAML bytes to JSON bytes.
func YamlToJSON(yamlBytes []byte) ([]byte, error) {
	var data any
	if err := yaml.Unmarshal(yamlBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
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

func MergeYaml(files ...string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no YAML files provided for merging")
	}

	var merged []byte
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", file, err)
		}

		merged, err = deepmerge.YAML(merged, content)
		if err != nil {
			return "", fmt.Errorf("failed to merge file %s: %w", file, err)
		}
	}

	return string(merged), nil
}
