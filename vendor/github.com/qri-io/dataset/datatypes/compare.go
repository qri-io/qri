package datatypes

import (
	"bytes"
	"fmt"
)

func CompareTypeBytes(a, b []byte, t Type) (int, error) {
	if len(a) == 0 && len(b) > 0 {
		return 1, nil
	} else if len(b) == 0 && len(a) > 0 {
		return -1, nil
	} else if len(b) == 0 && len(a) == 0 {
		return 0, nil
	}

	switch t {
	case String:
		return bytes.Compare(a, b), nil
	case Integer:
		return CompareIntegerBytes(a, b)
	case Float:
		return CompareFloatBytes(a, b)
	default:
		// TODO - other types
		return 0, fmt.Errorf("invalid type comparison")
	}
}

func CompareIntegerBytes(a, b []byte) (int, error) {
	at, err := ParseInteger(a)
	if err != nil {
		return 0, err
	}
	bt, err := ParseInteger(b)
	if err != nil {
		return 0, err
	}
	if at > bt {
		return 1, nil
	} else if at == bt {
		return 0, nil
	}
	return -1, nil
}

func CompareFloatBytes(a, b []byte) (int, error) {
	at, err := ParseFloat(a)
	if err != nil {
		return 0, err
	}
	bt, err := ParseFloat(b)
	if err != nil {
		return 0, err
	}
	if at > bt {
		return 1, nil
	} else if at == bt {
		return 0, nil
	}
	return -1, nil
}
