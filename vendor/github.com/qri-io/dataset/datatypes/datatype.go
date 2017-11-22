package datatypes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
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
	Json
)

const NUM_DATA_TYPES = 11

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
		Json:    "json",
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
		"json":    Json,
	}[t]
	if !ok {
		return Unknown
	}

	return got
}

// MarshalJSON implements json.Marshaler on Type
func (dt Type) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, dt.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler on Type
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
	if _, err = ParseJson(value); err == nil {
		return Json
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
	case Url:
		parsed, err = ParseUrl(value)
	case Json:
		parsed, err = ParseJson(value)
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
	case Json:
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
	return strconv.ParseFloat(string(value), 64)
}

func ParseInteger(value []byte) (int64, error) {
	return strconv.ParseInt(string(value), 10, 64)
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

func JsonArrayOrObject(value []byte) (string, error) {
	obji := bytes.IndexRune(value, '{')
	arri := bytes.IndexRune(value, '[')
	if obji == -1 && arri == -1 {
		return "", fmt.Errorf("invalid json data")
	}
	if (obji < arri || arri == -1) && obji >= 0 {
		return "object", nil
	}
	return "array", nil
}

func ParseJson(value []byte) (interface{}, error) {
	t, err := JsonArrayOrObject(value)
	if err != nil {
		return nil, err
	}

	if t == "object" {
		p := map[string]interface{}{}
		err = json.Unmarshal(value, &p)
		return p, err
	}

	p := []interface{}{}
	err = json.Unmarshal(value, &p)
	return p, err
}
