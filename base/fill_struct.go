package base

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// FillStruct fills in the values of an arbitrary structure using an already deserialized
// map of nested data. Fields names are case-insensitive. Unknown fields are treated as an
// error, *unless* the output structure implementes the KeyValSetter interface.
func FillStruct(fields map[string]interface{}, output interface{}) error {
	target := reflect.ValueOf(output)
	if target.Kind() == reflect.Ptr {
		target = target.Elem()
	}
	return putFieldsToTargetStruct(fields, target)
}

// KeyValSetter should be implemented by structs that can store arbitrary fields in a private map.
type KeyValSetter interface {
	SetKeyVal(string, interface{}) error
}
// TODO (dlong): Implement this interface for dataset.Meta. It currently has the similar method
// `Set`, which does more than needed, since it assigns to any field, not just the private map.

// timeObj is used for reflect.TypeOf
var timeObj time.Time

// putFieldsToTargetStruct iterates over the fields in the target struct, and assigns each
// field the value from the `fields` map. Recursively call this for an sub structures. Field
// names are treated as case-insensitive. Return any errors found during this process, or nil if
// there are no errors.
func putFieldsToTargetStruct(fields map[string]interface{}, target reflect.Value) error {
	if target.Kind() != reflect.Struct {
		return fmt.Errorf("can only put fields to a struct")
	}

	// Collect errors that occur in this single call, at the end of the function, join them
	// using newlines, if any exist.
	errs := make([]string, 0)

	// Collect real key names used by the `fields` map.
	realKeys := make([]string, 0)
	for k := range fields {
		realKeys = append(realKeys, k)
	}
	// Handle case-insensitivity by building a map from lowercase keys to real keys.
	caseMap := make(map[string]string)
	for i := 0; i < len(realKeys); i++ {
		realKey := realKeys[i]
		lowerKey := strings.ToLower(realKey)
		caseMap[lowerKey] = realKey
	}

	// Keep track of which keys have been used from the `fields` map
	usedKeys := make(map[string]bool)

	for i := 0; i < target.NumField(); i++ {
		// Lowercase the key in order to make matching case-insensitive.
		fieldName := target.Type().Field(i).Name
		lowerName := strings.ToLower(fieldName)

		val := fields[caseMap[lowerName]]
		if val == nil {
			// Nothing to assign to this field, go to next.
			continue
		}
		usedKeys[caseMap[lowerName]] = true

		// Dispatch on kind of field.
		field := target.Field(i)
		switch field.Kind() {
		case reflect.Int:
			num, ok := val.(int)
			if ok {
				field.SetInt(int64(num))
				continue
			}
			errs = append(errs, fmt.Sprintf("field %s type int, value %s", fieldName, val))
		case reflect.String:
			text, ok := val.(string)
			if ok {
				field.SetString(text)
				continue
			}
			errs = append(errs, fmt.Sprintf("field %s type string, value %s", fieldName, val))
		case reflect.Struct:
			// Specially handle time.Time, represented as a string, which needs to be parsed.
			if field.Type() == reflect.TypeOf(timeObj) {
				timeText, ok := val.(string)
				if ok {
					ts, err := time.Parse(time.RFC3339, timeText)
					if err == nil {
						field.Set(reflect.ValueOf(ts))
						continue
					}
					errs = append(errs, err.Error())
					continue
				}
				errs = append(errs, fmt.Sprintf("field %s type time, value %s", fieldName, val))
				continue
			}
			// Other struct types are not handled currently. Should probably do the same thing
			// as what's done for `pointer` below.
			errs = append(errs, fmt.Sprintf("unknown struct %s for field %s\n", field.Type(), fieldName))
		case reflect.Map:
			m, ok := val.(map[string]interface{})
			if ok {
				field.Set(reflect.ValueOf(m))
				continue
			}
			errs = append(errs, fmt.Sprintf("field %s type map, value %s", fieldName, val))
		case reflect.Ptr:
			// Allocate a new pointer for the sub-component to be filled in.
			alloc := reflect.New(field.Type().Elem())
			field.Set(alloc)
			inner := alloc.Elem()
			// For now, can only point to a struct.
			if inner.Kind() != reflect.Struct {
				errs = append(errs, fmt.Sprintf("can only assign to *struct @ %s", fieldName))
				continue
			}
			// Struct must be assigned from a map.
			component, err := toStringMap(val)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			// Recursion to handle sub-component.
			err = putFieldsToTargetStruct(component, inner)
			if err != nil {
				errs = append(errs, err.Error())
			}
		default:
			errs = append(errs, fmt.Sprintf("unknown kind %s, field name %s (IMPLEMENT ME)", field.Kind(), fieldName))
		}
	}

	// If the target struct is able, assign unknown keys to it.
	arbitrarySetter := getArbitraryKeyValSetter(target)

	// Iterate over keys in the `fields` data, see if there were any keys that were not stored in
	// the target struct.
	for i := 0; i < len(realKeys); i++ {
		k := realKeys[i]
		if _, ok := usedKeys[k]; !ok {
			// If target struct allows storing unknown keys to a map of arbitrary data.
			if arbitrarySetter != nil {
				arbitrarySetter.SetKeyVal(k, fields[k])
				continue
			}
			// Otherwise, unknown fields are an error.
			errs = append(errs, fmt.Sprintf("field \"%s\" not found in target", k))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errs, "\n"))
}

// toStringMap converts the input to a map[string] if able. This is needed because, while JSON
// correctly deserializes sub structures to map[string], YAML instead deserializes to
// map[interface{}]interface{}, so we need to manually convert this case to map[string].
func toStringMap(val interface{}) (map[string]interface{}, error) {
	m, ok := val.(map[string]interface{})
	if ok {
		return m, nil
	}
	imap, ok := val.(map[interface{}]interface{})
	if ok {
		convert := make(map[string]interface{})
		for k, v := range imap {
			convert[fmt.Sprintf("%v", k)] = v
		}
		return convert, nil
	}
	return nil, fmt.Errorf("could not convert to map[string]")
}

// getArbitraryKeyValSetter returns a KeyValSetter if the target implements it.
func getArbitraryKeyValSetter(target reflect.Value) KeyValSetter {
	if !target.CanAddr() {
		return nil
	}
	ptr := target.Addr()
	iface := ptr.Interface()
	if s, ok := iface.(KeyValSetter); ok {
		return s
	}
	return nil
}
