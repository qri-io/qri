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

// FieldError represents an error when a value does not match the expected type
type FieldError struct {
	Got  string
	Want string
	Val  interface{}
}

// Error displays the text version of the FieldError
func (c *FieldError) Error() string {
	return fmt.Sprintf("need %s, got %s: %#v", c.Want, c.Got, c.Val)
}

// SetPathValue sets a value on a the output struct, accessed using the path of dot-separated fields
func SetPathValue(path string, val interface{}, output interface{}) error {
	target := reflect.ValueOf(output)
	steps := strings.Split(path, ".")
	// Collector errors
	collector := NewErrorCollector()
	for _, s := range steps {
		collector.PushField(s)
	}
	// Find target place to assign to, field is a non-empty string only if target is a map.
	target, field, err := findTargetAtPath(steps, target)
	if err != nil {
		if err == ErrNotFound {
			return fmt.Errorf("at %q: %w", path, ErrNotFound)
		}
		collector.Add(err)
		return collector.AsSingleError()
	}
	if field == "" {
		// Convert val into the correct type, parsing ints and bools and so forth.
		val, err = coerceToTargetType(val, target)
		if err != nil {
			collector.Add(err)
			return collector.AsSingleError()
		}
		putValueToPlace(val, target, collector)
		return collector.AsSingleError()
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
		return nil, fmt.Errorf("at %q: %w", path, err)
	}
	// If all steps completed, resolve the target to a real value.
	if field == "" {
		return target.Interface(), nil
	}
	// Otherwise, one step is left, look it up using case-insensitive comparison.
	if lookup := mapLookupCaseInsensitive(target, field); lookup != nil {
		return lookup.Interface(), nil
	}
	return nil, fmt.Errorf("at %q: %w", path, ErrNotFound)
}

// Recursively look up the first step in place, until there are no steps left. If place is ever
// a map with one step left, return that map and final step - this is done in order to enable
// assignment to that map.
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
	} else if place.Kind() == reflect.Interface {
		var inner reflect.Value
		if place.IsNil() {
			alloc := reflect.New(place.Type().Elem())
			place.Set(alloc)
			inner = alloc.Elem()
		} else {
			inner = place.Elem()
		}
		return findTargetAtPath(steps, inner)
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
		if len(steps) == 1 {
			return place, s, nil
		}
		found := mapLookupCaseInsensitive(place, s)
		if found == nil {
			return place, "", ErrNotFound
		}
		return findTargetAtPath(rest, *found)
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

// Look up value in the map, using case-insensitive string comparison. Return nil if not found
func mapLookupCaseInsensitive(mapValue reflect.Value, k string) *reflect.Value {
	// Try looking up the value without changing the string case
	key := reflect.ValueOf(k)
	result := mapValue.MapIndex(key)
	if result.IsValid() {
		return &result
	}
	// Try lower-casing the key and looking that up
	klower := strings.ToLower(k)
	key = reflect.ValueOf(klower)
	result = mapValue.MapIndex(key)
	if result.IsValid() {
		return &result
	}
	// Iterate over the map keys, compare each, using case-insensitive matching
	mapKeys := mapValue.MapKeys()
	for _, mk := range mapKeys {
		mstr := mk.String()
		if strings.ToLower(mstr) == klower {
			result = mapValue.MapIndex(mk)
			if result.IsValid() {
				return &result
			}
		}
	}
	// Not found, return nil
	return nil
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
			}
		}
		b, ok := val.(bool)
		if ok {
			return b, nil
		}
		return nil, &FieldError{Want: "bool", Got: reflect.TypeOf(val).Name(), Val: val}
	case reflect.Int:
		num, ok := val.(int)
		if ok {
			return num, nil
		}
		str, ok := val.(string)
		if ok {
			parsed, err := strconv.ParseInt(str, 10, 32)
			if err == nil {
				return int(parsed), nil
			}
		}
		return nil, &FieldError{Want: "int", Got: reflect.TypeOf(val).Name(), Val: val}
	case reflect.Int64:
		num, ok := val.(int)
		if ok {
			return num, nil
		}
		num64, ok := val.(int64)
		if ok {
			return num64, nil
		}
		float64, ok := val.(float64)
		if ok {
			return float64, nil
		}
		str, ok := val.(string)
		if ok {
			parsed, err := strconv.ParseInt(str, 10, 64)
			if err == nil {
				return int64(parsed), nil
			}
		}
		return nil, &FieldError{Want: "int64", Got: reflect.TypeOf(val).Name(), Val: val}
	case reflect.Ptr:
		alloc := reflect.New(place.Type().Elem())
		return coerceToTargetType(val, alloc.Elem())
	case reflect.String:
		str, ok := val.(string)
		if ok {
			return str, nil
		}
		return nil, &FieldError{Want: "string", Got: reflect.TypeOf(val).Name(), Val: val}
	default:
		return nil, fmt.Errorf("unknown kind: %s", place.Kind())
	}
}

func coerceToInt(str string) (int, error) {
	parsed, err := strconv.ParseInt(str, 10, 32)
	if err == nil {
		return int(parsed), nil
	}
	return -1, &FieldError{Want: "int", Got: reflect.TypeOf(str).Name(), Val: str}
}

func getFieldCaseInsensitive(place reflect.Value, name string) reflect.Value {
	name = strings.ToLower(name)
	return place.FieldByNameFunc(func(s string) bool { return strings.ToLower(s) == name })
}
