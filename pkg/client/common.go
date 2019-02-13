package client

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

func doClean(in interface{}) interface{} {
	mapResult, ok := in.(map[interface{}]interface{})
	if ok {
		result := make(map[interface{}]interface{})
		for k, v := range mapResult {
			key := strings.ToLower(fmt.Sprint(k))
			if strings.HasPrefix(key, "xxx_") {
				delete(mapResult, key)
				continue
			}
			v = doClean(v)
			if v == nil {
				continue
			}
			result[k] = v

		}
		if len(result) == 0 {
			return nil
		}
		if len(result) == 1 {
			for _, skipKey := range []string{"kind", "fields", "structvalue", "string_value", "attributes", "stringvalue", "boolvalue", "listvalue"} {
				if result[skipKey] != nil {
					return result[skipKey]
				}
			}
		}
		return result
	}
	listResult, ok := in.([]interface{})
	if ok {
		var result []interface{}
		for _, elem := range listResult {
			result = append(result, doClean(elem))
		}
		if len(result) == 0 {
			return nil
		}
		return result
	}
	return in
}

func DoPrint(in interface{}) {
	yaml_data, _ := yaml.Marshal(in)
	data := make(map[interface{}]interface{})
	yaml.Unmarshal(yaml_data, data)

	yaml_data, _ = yaml.Marshal(doClean(data))
	fmt.Printf("%s\n", yaml_data)

}
