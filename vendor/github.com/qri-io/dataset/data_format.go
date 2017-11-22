// TODO - consider placing this in a subpackage: dataformats
package dataset

import (
	"encoding/json"
	"fmt"
)

var ErrUnknownDataFormat = fmt.Errorf("Unknown Data Format")

// DataFormat represents different types of data
type DataFormat int

const (
	UnknownDataFormat DataFormat = iota
	CsvDataFormat
	JsonDataFormat
	XmlDataFormat
	XlsDataFormat
	CdxjDataFormat
	// TODO - make this list more exhaustive
)

// String implements stringer interface for DataFormat
func (f DataFormat) String() string {
	s, ok := map[DataFormat]string{
		UnknownDataFormat: "",
		CsvDataFormat:     "csv",
		JsonDataFormat:    "json",
		XmlDataFormat:     "xml",
		XlsDataFormat:     "xls",
		CdxjDataFormat:    "cdxj",
	}[f]

	if !ok {
		return ""
	}

	return s
}

// ParseDataFormatString takes a string representation of a data format
func ParseDataFormatString(s string) (df DataFormat, err error) {
	df, ok := map[string]DataFormat{
		"":      UnknownDataFormat,
		".csv":  CsvDataFormat,
		"csv":   CsvDataFormat,
		".json": JsonDataFormat,
		"json":  JsonDataFormat,
		".xml":  XmlDataFormat,
		"xml":   XmlDataFormat,
		".xls":  XlsDataFormat,
		"xls":   XlsDataFormat,
		".cdxj": CdxjDataFormat,
		"cdxj":  CdxjDataFormat,
	}[s]
	if !ok {
		err = fmt.Errorf("invalid data format: `%s`", s)
		df = UnknownDataFormat
	}

	return
}

// MarshalJSON satisfies the json.Marshaler interface
func (f DataFormat) MarshalJSON() ([]byte, error) {
	if f == UnknownDataFormat {
		return nil, ErrUnknownDataFormat
	}
	return []byte(fmt.Sprintf(`"%s"`, f.String())), nil
}

// UnmarshalJSON satisfies the json.Unmarshaler interface
func (f *DataFormat) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("Data Format type should be a string, got %s", data)
	}

	if df, err := ParseDataFormatString(s); err != nil {
		return err
	} else {
		*f = df
	}

	return nil
}
