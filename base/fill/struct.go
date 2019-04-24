// Package fill matches arbitrary values to struct fields using reflection
// fill is case-insensitive.
// It's primary use is to support decoding data
// from a number of serialization formats (JSON,YAML,CBOR) into an intermediate
// map[string]interface{} value which can then be used to "fill" arbitrary struct
// values
package fill

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Struct fills in the values of an arbitrary structure using an already deserialized
// map of nested data. Fields names are case-insensitive. Unknown fields are treated as an
// error, *unless* the output structure implements the ArbitrarySetter interface.
func Struct(fields map[string]interface{}, output interface{}) error {
	target := reflect.ValueOf(output)
	if target.Kind() == reflect.Ptr {
		target = target.Elem()
	}
	collector := NewErrorCollector()
	putFieldsToTargetStruct(fields, target, collector)
	return collector.AsSingleError()
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
func putFieldsToTargetStruct(fields map[string]interface{}, target reflect.Value, collector *ErrorCollector) {
	if target.Kind() != reflect.Struct {
		collector.Add(fmt.Errorf("can only put fields to a struct"))
		return
	}

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
		if val == nil {
			// Don't try and assign a nil value.
			continue
		}

		collector.PushField(fieldName)
		putValueToPlace(val, target.Field(i), collector)
		collector.PopField()
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
			collector.Add(fmt.Errorf("path \"%s\": not found in destination struct", k))
		}
	}
}

// putValueToPlace stores the val at the place, recusively if necessary
func putValueToPlace(val interface{}, place reflect.Value, collector *ErrorCollector) {
	switch place.Kind() {
	case reflect.Struct:
		// Specially handle time.Time, represented as a string, which needs to be parsed.
		if place.Type() == reflect.TypeOf(timeObj) {
			timeText, ok := val.(string)
			if ok {
				ts, err := time.Parse(time.RFC3339, timeText)
				if err != nil {
					err = fmt.Errorf("could not parse time: \"%s\"", timeText)
					collector.Add(err)
				} else {
					place.Set(reflect.ValueOf(ts))
				}
				return
			}
			err := &FieldError{Want: "time", Got: reflect.TypeOf(val).Name(), Val: val}
			collector.Add(err)
			return
		}
		// Struct must be assigned from a map.
		component, err := toStringMap(val)
		if collector.Add(err) {
			return
		}
		// Recursion to handle sub-component.
		putFieldsToTargetStruct(component, place, collector)
	case reflect.Map:
		if val == nil {
			// If map is nil, nothing more to do.
			return
		}
		m, ok := val.(map[string]interface{})
		if !ok {
			collector.Add(&FieldError{Want: "map", Got: reflect.TypeOf(val).Name(), Val: val})
			return
		}
		// Special case map[string]string, convert values to strings.
		if place.Type().Elem() == reflect.TypeOf(strObj) {
			strmap := make(map[string]string)
			for k, v := range m {
				strmap[k] = fmt.Sprintf("%s", v)
			}
			place.Set(reflect.ValueOf(strmap))
			return
		}
		place.Set(reflect.ValueOf(m))
		return
	case reflect.Slice:
		if val == nil {
			// If slice is nil, nothing more to do.
			return
		}
		slice, ok := val.([]interface{})
		if !ok {
			collector.Add(fmt.Errorf("need type slice, value %s", val))
			return
		}
		// Get size of type of the slice to deserialize.
		size := len(slice)
		sliceType := place.Type().Elem()
		// Construct a new, empty slice of the same size.
		create := reflect.MakeSlice(reflect.SliceOf(sliceType), size, size)
		// Fill in each element.
		for i := 0; i < size; i++ {
			elem := reflect.Indirect(reflect.New(sliceType))
			collector.PushField(fmt.Sprintf("%d", i))
			putValueToPlace(slice[i], elem, collector)
			collector.PopField()
			create.Index(i).Set(elem)
		}
		place.Set(create)
		return
	case reflect.Array:
		if val == nil {
			// If slice is nil, nothing more to do.
			return
		}
		slice, ok := val.([]interface{})
		if !ok {
			collector.Add(fmt.Errorf("need type slice, value %s", val))
			return
		}
		// Get size of type of the slice to deserialize.
		size := len(slice)
		targetElem := place.Type().Elem()
		targetSize := place.Type().Len()
		if size != targetSize {
			collector.Add(fmt.Errorf("need array of size %d, got size %d", targetSize, size))
			return
		}
		// Construct array of appropriate size and type.
		arrayType := reflect.ArrayOf(targetSize, targetElem)
		create := reflect.New(arrayType).Elem()
		// Fill in each element.
		for i := 0; i < size; i++ {
			elem := reflect.Indirect(reflect.New(targetElem))
			putValueToPlace(slice[i], elem, collector)
			create.Index(i).Set(elem)
		}
		place.Set(create)
		return
	case reflect.Ptr:
		if val == nil {
			// If pointer is nil, nothing more to do.
			return
		}
		// Allocate a new pointer for the sub-component to be filled in.
		alloc := reflect.New(place.Type().Elem())
		place.Set(alloc)
		inner := alloc.Elem()
		putValueToPlace(val, inner, collector)
		return
	default:
		collector.Add(putValueToUnit(val, place))
		return
	}
}

// putValueToUnit stores the val at the place, as long as it is a unitary (non-compound) type
func putValueToUnit(val interface{}, place reflect.Value) error {
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
		return &FieldError{Want: "int", Got: reflect.TypeOf(val).Name(), Val: val}
	case reflect.Int64:
		num, ok := val.(int)
		if ok {
			place.SetInt(int64(num))
			return nil
		}
		num64, ok := val.(int64)
		if ok {
			place.SetInt(num64)
			return nil
		}
		float64, ok := val.(float64)
		if ok {
			place.SetInt(int64(float64))
			return nil
		}
		return &FieldError{Want: "int64", Got: reflect.TypeOf(val).Name(), Val: val}
	case reflect.Uint64:
		num, ok := val.(uint)
		if ok {
			place.SetUint(uint64(num))
			return nil
		}
		num64, ok := val.(uint64)
		if ok {
			place.SetUint(num64)
			return nil
		}
		float64, ok := val.(float64)
		if ok {
			place.SetUint(uint64(float64))
			return nil
		}
		return &FieldError{Want: "uint64", Got: reflect.TypeOf(val).Name(), Val: val}
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
		return &FieldError{Want: "float64", Got: reflect.TypeOf(val).Name(), Val: val}
	case reflect.String:
		text, ok := val.(string)
		if ok {
			place.SetString(text)
			return nil
		}
		return &FieldError{Want: "string", Got: reflect.TypeOf(val).Name(), Val: val}
	case reflect.Bool:
		b, ok := val.(bool)
		if ok {
			place.SetBool(b)
			return nil
		}
		return &FieldError{Want: "bool", Got: reflect.TypeOf(val).Name(), Val: val}
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
