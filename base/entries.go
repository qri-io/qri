package base

import (
	"fmt"

	"github.com/qri-io/dataset/dsio"
)

// ReadEntries reads entries and returns them as a native go array or map
func ReadEntries(reader dsio.EntryReader, all bool, limit int, offset int) (interface{}, error) {
	obj := make(map[string]interface{})
	array := make([]interface{}, 0)
	numRead := 0

	tlt, err := dsio.GetTopLevelType(reader.Structure())
	if err != nil {
		return nil, err
	}

	for i := 0;; i++ {
		val, err := reader.ReadEntry()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		if !all && i < offset {
			continue
		}

		if tlt == "object" {
			obj[val.Key] = val.Value
		} else {
			array = append(array, val.Value)
		}

		numRead++
		if !all && numRead == limit {
			break
		}
	}

	if tlt == "object" {
		return obj, nil
	}
	return array, nil
}

// ReadEntriesToArray reads entries and returns them as a native go array
func ReadEntriesToArray(reader dsio.EntryReader, all bool, limit int, offset int) ([]interface{}, error) {
	entries, err := ReadEntries(reader, all, limit, offset)
	if err != nil {
		return nil, err
	}

	array, ok := entries.([]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot convert top-level to array")
	}

	return array, nil
}
