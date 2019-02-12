package base

import (
	"github.com/qri-io/dataset/dsio"
)

// ReadEntries reads entries and returns them as a native go array or map
func ReadEntries(reader dsio.EntryReader) (interface{}, error) {
	obj := make(map[string]interface{})
	array := make([]interface{}, 0)

	tlt, err := dsio.GetTopLevelType(reader.Structure())
	if err != nil {
		return nil, err
	}

	for i := 0; ; i++ {
		val, err := reader.ReadEntry()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		if tlt == "object" {
			obj[val.Key] = val.Value
		} else {
			array = append(array, val.Value)
		}
	}

	if tlt == "object" {
		return obj, nil
	}
	return array, nil
}
