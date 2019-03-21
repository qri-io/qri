package fill

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var (
	// ErrNotFound is returned when a field isn't found
	ErrNotFound = fmt.Errorf("not found")
)

// CoercionError represents an error when a value cannot be converted from one type to another
type CoercionError struct {
	From string
	To   string
}

// Error displays the text version of the CoercionError
func (c *CoercionError) Error() string {
	return fmt.Sprintf("could not coerce from %s to %s", c.From, c.To)
}

// SetPathValue sets a value on a the output struct, accessed using the path of dot-separated fields
func SetPathValue(path string, val interface{}, output interface{}) error {
	target := reflect.ValueOf(output)
	steps := strings.Split(path, ".")
	// Find target place to assign to, field is a non-empty string only if target is a map.
	target, field, err := findTargetAtPath(steps, target)
	if err != nil {
		if err == ErrNotFound {
			return fmt.Errorf("path: \"%s\" not found", path)
		}
		return err
	}
	if field == "" {
		// Convert val into the correct type, parsing ints and bools and so forth.
		val, err = coerceToTargetType(val, target)
		if err != nil {
			if cerr, ok := err.(*CoercionError); ok {
				return fmt.Errorf("invalid type for path \"%s\": expected %s, got %s",
					path, cerr.To, cerr.From)
			}
			return err
		}
		return putValueToPlace(val, target)
	}
	// A map with a field name to assign to.
	// TODO: Only works for map[string]string, not map's with a struct for a value.
	target.SetMapIndex(reflect.ValueOf(field), reflect.ValueOf(val))
	return nil
}

// GetPathValue gets a value from the input struct, accessed using the path of dot-separated fields
func GetPathValue(path string, input interface{}) (interface{}, error) {
	target := reflect.ValueOf(input)
	steps := strings.Split(path, ".")
	// Find target place to assign to, field is a non-empty string only if target is a map.
	target, field, err := findTargetAtPath(steps, target)
	if err != nil {
		if err == ErrNotFound {
			return nil, fmt.Errorf("path: \"%s\" not found", path)
		}
		return nil, err
	}
	if field == "" {
		return target.Interface(), nil
	}
	lookup := target.MapIndex(reflect.ValueOf(field))
	if lookup.IsValid() {
		return lookup.Interface(), nil
	}
	return nil, fmt.Errorf("invalid path: \"%s\"", path)
}

func findTargetAtPath(steps []string, place reflect.Value) (reflect.Value, string, error) {
	if len(steps) == 0 {
		return place, "", nil
	}
	// Get current step of the path, dispatch on its type.
	s := steps[0]
	rest := steps[1:]
	if place.Kind() == reflect.Struct {
		field := getFieldCaseInsensitive(place, s)
		if !field.IsValid() {
			return place, "", ErrNotFound
		}
		return findTargetAtPath(rest, field)
	} else if place.Kind() == reflect.Ptr {
		var inner reflect.Value
		if place.IsNil() {
			alloc := reflect.New(place.Type().Elem())
			place.Set(alloc)
			inner = alloc.Elem()
		} else {
			inner = place.Elem()
		}
		return findTargetAtPath(steps, inner)
	} else if place.Kind() == reflect.Map {
		if place.IsNil() {
			place.Set(reflect.MakeMap(place.Type()))
		}
		// TODO: Handle case where `rest` has more steps and `val` is a struct: more
		// recursive is needed.
		return place, s, nil
	} else if place.Kind() == reflect.Slice {
		num, err := coerceToInt(s)
		if err != nil {
			return place, "", err
		}
		if num >= place.Len() {
			return place, "", fmt.Errorf("index outside of range: %d, len is %d", num, place.Len())
		}
		elem := place.Index(num)
		return findTargetAtPath(rest, elem)
	} else {
		return place, "", fmt.Errorf("cannot set field of type %s", place.Kind())
	}
}

func coerceToTargetType(val interface{}, place reflect.Value) (interface{}, error) {
	switch place.Kind() {
	case reflect.Bool:
		str, ok := val.(string)
		if ok {
			str = strings.ToLower(str)
			if str == "true" {
				return true, nil
			} else if str == "false" {
				return false, nil
			} else {
				return nil, fmt.Errorf("could not parse value to bool")
			}
		}
		b, ok := val.(bool)
		if ok {
			return b, nil
		}
		return nil, &CoercionError{To: "bool", From: reflect.TypeOf(val).Name()}
	case reflect.Int:
		num, ok := val.(int)
		if ok {
			return num, nil
		}
		str, ok := val.(string)
		if ok {
			parsed, err := strconv.ParseInt(str, 10, 32)
			return int(parsed), err
		}
		return nil, &CoercionError{To: "int", From: reflect.TypeOf(val).Name()}
	case reflect.Ptr:
		alloc := reflect.New(place.Type().Elem())
		return coerceToTargetType(val, alloc.Elem())
	case reflect.String:
		str, ok := val.(string)
		if ok {
			return str, nil
		}
		return nil, &CoercionError{To: "string", From: reflect.TypeOf(val).Name()}
	default:
		return nil, fmt.Errorf("unknown kind: %s", place.Kind())
	}
}

func coerceToInt(val interface{}) (int, error) {
	num, ok := val.(int)
	if ok {
		return num, nil
	}
	str, ok := val.(string)
	if ok {
		parsed, err := strconv.ParseInt(str, 10, 32)
		return int(parsed), err
	}
	return -1, &CoercionError{To: "int", From: reflect.TypeOf(val).Name()}
}

func getFieldCaseInsensitive(place reflect.Value, name string) reflect.Value {
	name = strings.ToLower(name)
	return place.FieldByNameFunc(func(s string) bool { return strings.ToLower(s) == name })
}
