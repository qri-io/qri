package dsutil

import (
	"fmt"

	"github.com/qri-io/dataset"
	"gopkg.in/yaml.v2"
)

// UnmarshalYAMLDataset reads yaml bytes into a Dataset, dealing with the issue that
// YAML likes to unmarshal unknown values to map[interface{}]interface{} instead of
// map[string]interface{}
func UnmarshalYAMLDataset(data []byte, ds *dataset.Dataset) error {
	if err := yaml.Unmarshal(data, ds); err != nil {
		return err
	}
	if ds.Structure != nil && ds.Structure.Schema != nil {
		for key, val := range ds.Structure.Schema {
			ds.Structure.Schema[key] = cleanupMapValue(val)
		}
	}
	if ds.Transform != nil && ds.Transform.Config != nil {
		for key, val := range ds.Transform.Config {
			ds.Transform.Config[key] = cleanupMapValue(val)
		}
	}
	return nil
}

func cleanupInterfaceArray(in []interface{}) []interface{} {
	res := make([]interface{}, len(in))
	for i, v := range in {
		res[i] = cleanupMapValue(v)
	}
	return res
}

func cleanupInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range in {
		res[fmt.Sprintf("%v", k)] = cleanupMapValue(v)
	}
	return res
}

func cleanupMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanupInterfaceArray(v)
	case map[interface{}]interface{}:
		return cleanupInterfaceMap(v)
	case string, bool, int, int16, int32, int64, float32, float64, []byte:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
