package fill

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Struct fills in the values of an arbitrary structure using an already deserialized
// map of nested data. Fields names are case-insensitive. Unknown fields are treated as an
// error, *unless* the output structure implementes the ArbitrarySetter interface.
func Struct(fields map[string]interface{}, output interface{}) error {
	target := reflect.ValueOf(output)
	if target.Kind() == reflect.Ptr {
		target = target.Elem()
	}
	return putFieldsToTargetStruct(fields, target)
}

// ArbitrarySetter should be implemented by structs that can store arbitrary fields in a private map.
type ArbitrarySetter interface {
	SetArbitrary(string, interface{}) error
}

var (
	// timeObj and strObj are used for reflect.TypeOf
	timeObj time.Time
	strObj  string
)

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

		val, ok := fields[caseMap[lowerName]]
		if !ok {
			// Nothing to assign to this field, go to next.
			continue
		}
		usedKeys[caseMap[lowerName]] = true

		err := putValueToPlace(val, target.Field(i))
		if err != nil {
			errs = append(errs, fmt.Sprintf("field %s: %s", fieldName, err.Error()))
		}
	}

	// If the target struct is able, assign unknown keys to it.
	arbitrarySetter := getArbitrarySetter(target)

	// Iterate over keys in the `fields` data, see if there were any keys that were not stored in
	// the target struct.
	for i := 0; i < len(realKeys); i++ {
		k := realKeys[i]
		if _, ok := usedKeys[k]; !ok {
			// If target struct allows storing unknown keys to a map of arbitrary data.
			if arbitrarySetter != nil {
				arbitrarySetter.SetArbitrary(k, fields[k])
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

func putValueToPlace(val interface{}, place reflect.Value) error {
	switch place.Kind() {
	case reflect.Int:
		num, ok := val.(int)
		if ok {
			place.SetInt(int64(num))
			return nil
		}
		numFloat, ok := val.(float64)
		if ok {
			place.SetInt(int64(numFloat))
			return nil
		}
		return fmt.Errorf("need type int, value %s", val)
	case reflect.Float64:
		num, ok := val.(int)
		if ok {
			place.SetFloat(float64(num))
			return nil
		}
		numFloat, ok := val.(float64)
		if ok {
			place.SetFloat(numFloat)
			return nil
		}
		return fmt.Errorf("need type string, value %s", val)
	case reflect.String:
		text, ok := val.(string)
		if ok {
			place.SetString(text)
			return nil
		}
		return fmt.Errorf("need type string, value %s", val)
	case reflect.Bool:
		b, ok := val.(bool)
		if ok {
			place.SetBool(b)
			return nil
		}
		return fmt.Errorf("need type string, value %s", val)
	case reflect.Struct:
		// Specially handle time.Time, represented as a string, which needs to be parsed.
		if place.Type() == reflect.TypeOf(timeObj) {
			timeText, ok := val.(string)
			if ok {
				ts, err := time.Parse(time.RFC3339, timeText)
				if err == nil {
					place.Set(reflect.ValueOf(ts))
					return nil
				}
				return err
			}
			return fmt.Errorf("need type time, value %s", val)
		}
		// Struct must be assigned from a map.
		component, err := toStringMap(val)
		if err != nil {
			return err
		}
		// Recursion to handle sub-component.
		return putFieldsToTargetStruct(component, place)
	case reflect.Map:
		if val == nil {
			// If map is nil, nothing more to do.
			return nil
		}
		m, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("need type map, value %s", val)
		}
		// Special case map[string]string, convert values to strings.
		if place.Type().Elem() == reflect.TypeOf(strObj) {
			strmap := make(map[string]string)
			for k, v := range m {
				strmap[k] = fmt.Sprintf("%s", v)
			}
			place.Set(reflect.ValueOf(strmap))
			return nil
		}
		place.Set(reflect.ValueOf(m))
		return nil
	case reflect.Slice:
		if val == nil {
			// If slice is nil, nothing more to do.
			return nil
		}
		slice, ok := val.([]interface{})
		if !ok {
			return fmt.Errorf("need type slice, value %s", val)
		}
		// Get size of type of the slice to deserialize.
		size := len(slice)
		sliceType := place.Type().Elem()
		// Construct a new, empty slice of the same size.
		create := reflect.MakeSlice(reflect.SliceOf(sliceType), size, size)
		// Fill in each element.
		for i := 0; i < size; i++ {
			elem := reflect.Indirect(reflect.New(sliceType))
			err := putValueToPlace(slice[i], elem)
			if err != nil {
				return err
			}
			create.Index(i).Set(elem)
		}
		place.Set(create)
		return nil
	case reflect.Ptr:
		if val == nil {
			// If pointer is nil, nothing more to do.
			return nil
		}
		// Allocate a new pointer for the sub-component to be filled in.
		alloc := reflect.New(place.Type().Elem())
		place.Set(alloc)
		inner := alloc.Elem()
		err := putValueToPlace(val, inner)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown kind %s", place.Kind())
	}
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

// getArbitrarySetter returns a ArbitrarySetter if the target implements it.
func getArbitrarySetter(target reflect.Value) ArbitrarySetter {
	if !target.CanAddr() {
		return nil
	}
	ptr := target.Addr()
	iface := ptr.Interface()
	if setter, ok := iface.(ArbitrarySetter); ok {
		return setter
	}
	return nil
}
