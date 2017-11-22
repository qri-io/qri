package dataset

import (
	"encoding/json"
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/compression"
)

// Structure designates a deterministic definition for working with a discrete dataset.
// Structure is a concrete handle that provides precise details about how to interpret a given
// piece of data (the reference to the data itself is provided elsewhere, specifically in the dataset struct )
// These techniques provide mechanisms for joining & traversing multiple structures.
// This example is shown in a human-readable form, for storage on the network the actual
// output would be in a condensed, non-indented form, with keys sorted by lexographic order.
type Structure struct {
	// private storage for reference to this object
	path datastore.Key
	// Format specifies the format of the raw data MIME type
	Format DataFormat `json:"format"`
	// FormatConfig removes as much ambiguity as possible about how
	// to interpret the speficied format.
	FormatConfig FormatConfig `json:"formatConfig,omitempty"`
	// Encoding specifics character encoding
	// should assume utf-8 if not specified
	Encoding string `json:"encoding,omitempty"`
	// Compression specifies any compression on the source data,
	// if empty assume no compression
	Compression compression.Type `json:"compression,omitempty"`
	// Schema contains the schema definition for the underlying data
	Schema *Schema `json:"schema"`
}

func (s *Structure) Path() datastore.Key {
	return s.path
}

// NewStructureRef creates an empty struct with it's
// internal path set
func NewStructureRef(path datastore.Key) *Structure {
	return &Structure{path: path}
}

// Abstract returns this structure instance in it's "Abstract" form
// stripping all nonessential values &
// renaming all schema field names to standard variable names
func (s *Structure) Abstract() *Structure {
	a := &Structure{
		Format:       s.Format,
		FormatConfig: s.FormatConfig,
		Encoding:     s.Encoding,
	}
	if s.Schema != nil {
		a.Schema = &Schema{
			PrimaryKey: s.Schema.PrimaryKey,
			Fields:     make([]*Field, len(s.Schema.Fields)),
		}
		for i, f := range s.Schema.Fields {
			a.Schema.Fields[i] = &Field{
				Name:         AbstractColumnName(i),
				Type:         f.Type,
				MissingValue: f.MissingValue,
				Format:       f.Format,
				Constraints:  f.Constraints,
			}
		}
	}
	return a
}

// Hash gives the hash of this structure
func (r *Structure) Hash() (string, error) {
	return JSONHash(r)
}

// separate type for marshalling into & out of
// most importantly, struct names must be sorted lexographically
type _structure struct {
	Compression  compression.Type       `json:"compression,omitempty"`
	Encoding     string                 `json:"encoding,omitempty"`
	Format       DataFormat             `json:"format"`
	FormatConfig map[string]interface{} `json:"formatConfig,omitempty"`
	Schema       *Schema                `json:"schema,omitempty"`
}

// MarshalJSON satisfies the json.Marshaler interface
func (s Structure) MarshalJSON() (data []byte, err error) {
	if s.path.String() != "" && s.Encoding == "" && s.Schema == nil {
		return s.path.MarshalJSON()
	}

	var opt map[string]interface{}
	if s.FormatConfig != nil {
		opt = s.FormatConfig.Map()
	}

	return json.Marshal(&_structure{
		Compression:  s.Compression,
		Encoding:     s.Encoding,
		Format:       s.Format,
		FormatConfig: opt,
		Schema:       s.Schema,
	})
}

// UnmarshalJSON satisfies the json.Unmarshaler interface
func (s *Structure) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = Structure{path: datastore.NewKey(str)}
		return nil
	}

	_s := &_structure{}
	if err := json.Unmarshal(data, _s); err != nil {
		return fmt.Errorf("error unmarshaling dataset structure from json: %s", err.Error())
	}

	fmtCfg, err := ParseFormatConfigMap(_s.Format, _s.FormatConfig)
	if err != nil {
		return fmt.Errorf("error parsing structure formatConfig: %s", err.Error())
	}

	*s = Structure{
		Compression:  _s.Compression,
		Encoding:     _s.Encoding,
		Format:       _s.Format,
		FormatConfig: fmtCfg,
		Schema:       _s.Schema,
	}

	// TODO - question of weather we should not accept
	// invalid structure defs at parse time. For now we'll take 'em.
	// if err := d.Valid(); err != nil {
	//   return err
	// }

	return nil
}

func (st *Structure) IsEmpty() bool {
	return st.Format == UnknownDataFormat && st.FormatConfig == nil && st.Encoding == "" && st.Schema == nil
}

// Assign collapses all properties of a group of structures on to one
// this is directly inspired by Javascript's Object.assign
func (s *Structure) Assign(structures ...*Structure) {
	for _, st := range structures {
		if st == nil {
			continue
		}

		if st.path.String() != "" {
			s.path = st.path
		}
		if st.Format != UnknownDataFormat {
			s.Format = st.Format
		}
		if st.FormatConfig != nil {
			s.FormatConfig = st.FormatConfig
		}
		if st.Encoding != "" {
			s.Encoding = st.Encoding
		}
		if st.Compression != compression.None {
			s.Compression = st.Compression
		}

		if s.Schema == nil && st.Schema != nil {
			s.Schema = st.Schema
			continue
		}
		s.Schema.Assign(st.Schema)
	}
}

// StringFieldIndex gives the index of a field who's name matches s
// it returns -1 if no match is found
func (st *Structure) StringFieldIndex(s string) int {
	if st.Schema == nil {
		return -1
	}
	for i, f := range st.Schema.Fields {
		if f.Name == s {
			return i
		}
	}
	return -1
}

// UnmarshalStructure tries to extract a structure type from an empty
// interface. Pairs nicely with datastore.Get() from github.com/ipfs/go-datastore
func UnmarshalStructure(v interface{}) (*Structure, error) {
	switch r := v.(type) {
	case *Structure:
		return r, nil
	case Structure:
		return &r, nil
	case []byte:
		structure := &Structure{}
		err := json.Unmarshal(r, structure)
		return structure, err
	default:
		return nil, fmt.Errorf("couldn't parse structure")
	}
}

// AbstractTableName prepends a given index with "t"
// t1, t2, t3, ...
func AbstractTableName(i int) string {
	return fmt.Sprintf("t%d", i+1)
}

// AbstractColumnName is the "base26" value of a column name
// to make short, sql-valid, deterministic column names
func AbstractColumnName(i int) string {
	return base26(i)
}

// b26chars is a-z, lowercase
const b26chars = "abcdefghijklmnopqrstuvwxyz"

// base26 maps the set of natural numbers
// to letters, using repeating characters to handle values
// greater than 26
func base26(d int) (s string) {
	var cols []int
	if d == 0 {
		return "a"
	}

	for d != 0 {
		cols = append(cols, d%26)
		d = d / 26
	}
	for i := len(cols) - 1; i >= 0; i-- {
		if i != 0 && cols[i] > 0 {
			s += string(b26chars[cols[i]-1])
		} else {
			s += string(b26chars[cols[i]])
		}
	}
	return s
}

// CompareStructures checks if all fields of two structure pointers are equal,
// returning an error on the first mismatch, nil if equal
func CompareStructures(a, b *Structure) error {
	if a == nil && b == nil {
		return nil
	} else if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("Structure mismatch: %s != %s", a, b)
	}

	if err := CompareSchemas(a.Schema, b.Schema); err != nil {
		return fmt.Errorf("Schema mismatch: %s", err.Error())
	}

	return nil
}
