package datatypes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Type int

const (
	Unknown Type = iota
	Any
	String
	Integer
	Float
	Boolean
	Date
	Url
	Object
	Array
)

const NUM_DATA_TYPES = 9

func (dt Type) String() string {
	s, ok := map[Type]string{
		Unknown: "",
		Any:     "any",
		String:  "string",
		Integer: "integer",
		Float:   "float",
		Boolean: "boolean",
		Date:    "date",
		Url:     "url",
		Object:  "object",
		Array:   "array",
	}[dt]

	if !ok {
		return ""
	}

	return s
}

// TypeFromString takes a string & tries to return it's type
// defaulting to unknown if the type is unrecognized
func TypeFromString(t string) Type {
	got, ok := map[string]Type{
		"any":     Any,
		"string":  String,
		"integer": Integer,
		"float":   Float,
		"boolean": Boolean,
		"date":    Date,
		"url":     Url,
		"object":  Object,
		"array":   Array,
	}[t]
	if !ok {
		return Unknown
	}

	return got
}

func (dt Type) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, dt.String())), nil
}

func (dt *Type) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("Filed type should be a string, got %s", data)
	}

	t := TypeFromString(s)

	if t == Unknown {
		return fmt.Errorf("Unknown datatype '%s'", s)
	}

	*dt = t
	return nil
}

// TODO - should write a version of MUCH faster funcs with "Is" prefix (IsObject, etc)
// that just return t/f. these funcs should aim to bail ASAP when proven false
func ParseDatatype(value []byte) Type {
	var err error
	if _, err = ParseArray(value); err == nil {
		return Object
	}
	if _, err = ParseObject(value); err == nil {
		return Object
	}
	if _, err = ParseFloat(value); err == nil {
		return Float
	}
	if _, err = ParseInteger(value); err == nil {
		return Integer
	}
	if _, err = ParseBoolean(value); err == nil {
		return Boolean
	}
	if _, err = ParseDate(value); err == nil {
		return Date
	}
	// if _, err = ParseUrl(value); err == nil {
	// 	return Url
	// }
	// if _, err = ParseArray(value); err == nil {
	// 	return Array
	// }
	if _, err = ParseString(value); err == nil {
		return String
	}
	return Any
}

func (dt Type) Parse(value []byte) (parsed interface{}, err error) {
	switch dt {
	case Unknown:
		parsed, err = ParseUnknown(value)
	case Any:
		parsed, err = ParseAny(value)
	case String:
		parsed, err = ParseString(value)
	case Float:
		parsed, err = ParseFloat(value)
	case Integer:
		parsed, err = ParseInteger(value)
	case Boolean:
		parsed, err = ParseBoolean(value)
	case Date:
		parsed, err = ParseDate(value)
	case Object:
		parsed, err = ParseObject(value)
	case Url:
		parsed, err = ParseUrl(value)
	case Array:
		parsed, err = ParseArray(value)
	}
	return
}

// takes already-parsed values & converts them to a string
func (dt Type) ValueToString(value interface{}) (str string, err error) {
	switch dt {
	case Unknown:
		err = fmt.Errorf("cannot parse unknown value: %v", value)
		return
	case Any:
		// TODO
	case String:
		s, ok := value.(string)
		if !ok {
			err = fmt.Errorf("%v is not a %s value", value, dt.String())
			return
		}
		str = s
	case Integer:
		num, ok := value.(int)
		if !ok {
			err = fmt.Errorf("%v is not a %s value", value, dt.String())
			return
		}
		str = strconv.FormatInt(int64(num), 10)
	case Float:
		num, ok := value.(float32)
		if !ok {
			err = fmt.Errorf("%v is not a %s value", value, dt.String())
			return
		}
		str = strconv.FormatFloat(float64(num), 'g', -1, 64)
	case Boolean:
		val, ok := value.(bool)
		if !ok {
			err = fmt.Errorf("%v is not a %s value", value, dt.String())
			return
		}
		str = strconv.FormatBool(val)
	case Object, Array:
		data, e := json.Marshal(value)
		if e != nil {
			err = e
			return
		}
		str = string(data)
	case Date:
		val, ok := value.(time.Time)
		if !ok {
			err = fmt.Errorf("%v is not a %s value", value, dt.String())
			return
		}
		str = val.Format(time.RubyDate)
	case Url:
		val, ok := value.(*url.URL)
		if !ok {
			err = fmt.Errorf("%v is not a %s value", value, dt.String())
			return
		}
		str = val.String()
	}
	return
}

// takes already-parsed values & converts them to a slice of bytes
func (dt Type) ValueToBytes(value interface{}) (data []byte, err error) {
	// TODO - for now we just wrap ToString
	if str, err := dt.ValueToString(value); err != nil {
		return nil, err
	} else {
		data = []byte(str)
	}

	return
}

func ParseUnknown(value []byte) (interface{}, error) {
	return nil, errors.New("cannot parse unknown data type")
}

// TODO
func ParseAny(value []byte) (interface{}, error) {
	return nil, nil
}

func ParseString(value []byte) (string, error) {
	return string(value), nil
}

func ParseFloat(value []byte) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(string(value)), 64)
}

func ParseInteger(value []byte) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(string(value)), 10, 64)
}

func ParseBoolean(value []byte) (bool, error) {
	return strconv.ParseBool(string(value))
}

func ParseDate(value []byte) (t time.Time, err error) {
	str := string(value)
	for _, format := range []string{
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
	} {
		if t, err = time.Parse(format, str); err == nil {
			return
		}
	}
	return time.Now(), fmt.Errorf("invalid date: %s", str)
}

func ParseUrl(value []byte) (*url.URL, error) {
	if !Relaxed.Match(value) {
		return nil, fmt.Errorf("invalid url: %s", string(value))
	}
	return url.Parse(string(value))
}

// ParseObject assumes a json object
// WARNING - this may be an improper assumption
func ParseObject(value []byte) (object map[string]interface{}, err error) {
	err = json.Unmarshal(value, &object)
	return
}

// ParseArray assumes a json array
// WARNING - this may be an improper assumption
func ParseArray(value []byte) (array []interface{}, err error) {
	err = json.Unmarshal(value, &array)
	return
}
