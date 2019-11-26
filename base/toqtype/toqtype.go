package toqtype

// TODO(dlong): Utilities for creating qtype values. Will likely rename or move this soon.

import (
	"encoding/csv"
	"encoding/json"
	"strings"
)

// StructToMap converts any struct into a corresponding map with string keys for each field name
func StructToMap(value interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// MustParseJSONAsArray parses the string as a json array, or panics. Should only use in tests
func MustParseJSONAsArray(content string) []interface{} {
	var result []interface{}
	err := json.Unmarshal([]byte(content), &result)
	if err != nil {
		panic(err)
	}
	return result
}

// MustParseCsvAsArray parses the string as a json array, or panics. Should only use in tests
func MustParseCsvAsArray(content string) []interface{} {
	r := strings.NewReader(content)
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		panic(err)
	}
	result := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		result = append(result, row)
	}
	return result
}
