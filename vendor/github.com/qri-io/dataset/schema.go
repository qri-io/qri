package dataset

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset/datatypes"
)

// Schema is analogous to a SQL schema definition
type Schema struct {
	Fields     []*Field `json:"fields,omitempty"`
	PrimaryKey FieldKey `json:"primaryKey,omitempty"`
}

// FieldNames gives a slice of field names defined in schema
func (s *Schema) FieldNames() (names []string) {
	if s.Fields == nil {
		return []string{}
	}
	names = make([]string, len(s.Fields))
	for i, f := range s.Fields {
		names[i] = f.Name
	}
	return
}

// FieldForName returns the field who's string name matches name
func (s *Schema) FieldForName(name string) *Field {
	for _, f := range s.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// FieldTypeStrings gives a slice of each field's type as a string
func (s *Schema) FieldTypeStrings() (types []string) {
	types = make([]string, len(s.Fields))
	for i, f := range s.Fields {
		types[i] = f.Type.String()
	}
	return
}

// AssignSchema collapses all properties of a group of schemas on to one
// this is directly inspired by Javascript's Object.assign
func (s *Schema) Assign(schemas ...*Schema) {
	for _, sh := range schemas {
		if sh == nil {
			continue
		}

		// @TODO - wouldn't this be nice...
		// if s == nil && sh != nil {
		// 	s = sh
		// 	continue
		// }

		if sh.PrimaryKey != nil {
			s.PrimaryKey = sh.PrimaryKey
		}

		if s.Fields == nil && sh.Fields != nil {
			s.Fields = sh.Fields
			continue
		}

		for i, f := range sh.Fields {
			if i > len(s.Fields)-1 {
				s.Fields = append(s.Fields, f)
				continue
			}
			if s.Fields[i] == nil && f != nil {
				s.Fields[i] = f
				continue
			}
			s.Fields[i].Assign(f)
		}
	}
}

func (f *Field) Assign(fields ...*Field) {
	for _, fd := range fields {
		if fd == nil {
			continue
		}
		// TODO - wouldn't this be nice...
		// if f == nil && fd != nil {
		// 	f = fd
		// 	continue
		// }

		if fd.Name != "" {
			f.Name = fd.Name
		}
		if fd.Type != datatypes.Unknown {
			f.Type = fd.Type
		}
		if fd.MissingValue != nil {
			f.MissingValue = fd.MissingValue
		}
		if fd.Format != "" {
			f.Format = fd.Format
		}
		if fd.Constraints != nil {
			f.Constraints = fd.Constraints
		}
		if fd.Title != "" {
			f.Title = fd.Title
		}
		if fd.Description != "" {
			f.Description = fd.Description
		}
	}
}

// Field is a field descriptor
type Field struct {
	Name         string            `json:"name"`
	Type         datatypes.Type    `json:"type,omitempty"`
	MissingValue interface{}       `json:"missingValue,omitempty"`
	Format       string            `json:"format,omitempty"`
	Constraints  *FieldConstraints `json:"constraints,omitempty"`
	Title        string            `json:"title,omitempty"`
	Description  string            `json:"description,omitempty"`
}

// field is a private struct for marshaling into and out of JSON
// most importantly, keys are sorted by lexographical order
type _field struct {
	Constraints  *FieldConstraints `json:"constraints,omitempty"`
	Description  string            `json:"description,omitempty"`
	Format       string            `json:"format,omitempty"`
	MissingValue interface{}       `json:"missingValue,omitempty"`
	Name         string            `json:"name"`
	Title        string            `json:"title,omitempty"`
	Type         datatypes.Type    `json:"type,omitempty"`
}

// MarshalJSON satisfies the json.Marshaler interface
func (f Field) MarshalJSON() ([]byte, error) {
	_f := &_field{
		Constraints:  f.Constraints,
		Description:  f.Description,
		Format:       f.Format,
		MissingValue: f.MissingValue,
		Name:         f.Name,
		Title:        f.Title,
		Type:         f.Type,
	}
	return json.Marshal(_f)
}

// UnmarshalJSON satisfies the json.Unmarshaler interface
func (f *Field) UnmarshalJSON(data []byte) error {
	_f := &_field{}
	if err := json.Unmarshal(data, _f); err != nil {
		return fmt.Errorf("error unmarshaling dataset field from json: %s", err.Error())
	}

	*f = Field{
		Constraints:  _f.Constraints,
		Description:  _f.Description,
		Format:       _f.Format,
		MissingValue: _f.MissingValue,
		Name:         _f.Name,
		Title:        _f.Title,
		Type:         _f.Type,
	}
	return nil
}

// FieldKey allows a field key to be either a string or object
type FieldKey []string

// FieldConstraints is supposed to constrain the field,
// this is totally unfinished, unimplemented, and needs lots of work
// TODO - uh, finish this?
type FieldConstraints struct {
	Required  *bool         `json:"required,omitempty"`
	MinLength *int64        `json:"minLength,omitempty"`
	MaxLength *int64        `json:"maxLength,omitempty"`
	Unique    *bool         `json:"unique,omitempty"`
	Pattern   string        `json:"pattern,omitempty"`
	Minimum   interface{}   `json:"minimum,omitempty"`
	Maximum   interface{}   `json:"maximum,omitempty"`
	Enum      []interface{} `json:"enum,omitempty"`
}

// ForeignKey is supposed to be for supporting foreign key
// references. It's also totally unfinished.
// TODO - finish this
type ForeignKey struct {
	Fields FieldKey `json:"fields"`
	// Reference
}

// CompareSchemas checks if all fields of two Schema pointers are equal,
// returning an error on the first mismatch, nil if equal
func CompareSchemas(a, b *Schema) error {
	// if a.PrimaryKey != b.PrimaryKey {
	// 	return fmt.Errorf("primary key mismatch: %s != %s", a.PrimaryKey, b.PrimaryKey)
	// }
	if a == nil && b == nil {
		return nil
	}
	if a != nil && b == nil || a == nil && b != nil {
		return fmt.Errorf("nil mismatch: %v != %v", a, b)
	}
	if a.Fields == nil && b.Fields != nil || a.Fields != nil && b.Fields == nil {
		return fmt.Errorf("fields slice mismatch: %s != %s", a.Fields, b.Fields)
	}
	if a.Fields == nil && b.Fields == nil {
		return nil
	}

	if len(a.Fields) != len(b.Fields) {
		return fmt.Errorf("field length mismatch: %d != %d", len(a.Fields), len(b.Fields))
	}

	for i, af := range a.Fields {
		bf := b.Fields[i]
		if err := CompareFields(af, bf); err != nil {
			return fmt.Errorf("field %d mismatch: %s", i, err.Error())
		}
	}

	return nil
}

// CompareFields checks if all fields of two Field pointers are equal,
// returning an error on the first mismatch, nil if equal
func CompareFields(a, b *Field) error {
	if a == nil && b == nil {
		return nil
	} else if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("nil mismatch: %s != %s", a, b)
	}

	if a.Name != b.Name {
		return fmt.Errorf("name mismatch: %s != %s", a.Name, b.Name)
	}
	if a.Type != b.Type {
		return fmt.Errorf("field type mismatch: %s != %s", a.Type.String(), b.Type.String())
	}
	if a.Title != b.Title {
		return fmt.Errorf("title mismatch: %s != %s", a.Title, b.Title)
	}
	if a.Description != b.Description {
		return fmt.Errorf("description mismatch: %s != %s", a.Description, b.Description)
	}

	// TODO - finish comparison of field constraints, primary keys, format, etc.

	return nil
}
