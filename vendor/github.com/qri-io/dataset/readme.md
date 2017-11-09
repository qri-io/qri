# dataset
--
    import "github.com/qri-io/dataset"

TODO - consider placing this in a subpackage: dataformats

## Usage

```go
var ErrNoPath = fmt.Errorf("missing path")
```

```go
var ErrUnknownDataFormat = fmt.Errorf("Unknown Data Format")
```

#### func  HashBytes

```go
func HashBytes(data []byte) (hash string, err error)
```
HashBytes generates the base-58 encoded SHA-256 hash of a byte slice It's
important to note that this is *NOT* the same as an IPFS hash, These hash
functions should be used for other things like checksumming, in-memory
content-addressing, etc.

#### func  JSONHash

```go
func JSONHash(m json.Marshaler) (hash string, err error)
```
JSONHash calculates the hash of a json.Marshaler It's important to note that
this is *NOT* the same as an IPFS hash, These hash functions should be used for
other things like checksumming, in-memory content-addressing, etc.

#### type Citation

```go
type Citation struct {
	Name  string `json:"name,omitempty"`
	Url   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}
```

Citation is a place that this dataset drew it's information from

#### type CsvOptions

```go
type CsvOptions struct {
	// Weather this csv file has a header row or not
	HeaderRow bool `json:"headerRow"`
}
```


#### func (*CsvOptions) Format

```go
func (*CsvOptions) Format() DataFormat
```

#### func (*CsvOptions) Map

```go
func (o *CsvOptions) Map() map[string]interface{}
```

#### type DataFormat

```go
type DataFormat int
```

DataFormat represents different types of data

```go
const (
	UnknownDataFormat DataFormat = iota
	CsvDataFormat
	JsonDataFormat
	JsonArrayDataFormat
	XmlDataFormat
	XlsDataFormat
)
```

#### func  ParseDataFormatString

```go
func ParseDataFormatString(s string) (df DataFormat, err error)
```
ParseDataFormatString takes a string representation of a data format

#### func (DataFormat) MarshalJSON

```go
func (f DataFormat) MarshalJSON() ([]byte, error)
```
MarshalJSON satisfies the json.Marshaler interface

#### func (DataFormat) String

```go
func (f DataFormat) String() string
```
String implements stringer interface for DataFormat

#### func (*DataFormat) UnmarshalJSON

```go
func (f *DataFormat) UnmarshalJSON(data []byte) error
```
UnmarshalJSON satisfies the json.Unmarshaler interface

#### type Dataset

```go
type Dataset struct {

	// Time this dataset was created. Required. Datasets are immutable, so no "updated"
	Timestamp time.Time `json:"timestamp"`
	// Structure of this dataset, required
	Structure *Structure `json:"structure"`
	// AbstractStructure is the abstract form of the structure field
	AbstractStructure *Structure `json:"abstractStructure,omitempty"`

	// Data is the path to the hash of raw data as it resolves on the network.
	Data datastore.Key `json:"data"`
	// Length is the length of the data object in bytes.
	// must always match & be present
	Length int `json:"length"`
	// Previous connects datasets to form a historical DAG
	Previous datastore.Key `json:"previous,omitempty"`

	// Title of this dataset
	Title string `json:"title,omitempty"`
	// Url to access the dataset
	AccessUrl string `json:"accessUrl,omitempty"`
	// Url that should / must lead directly to the data itself
	DownloadUrl string `json:"downloadUrl,omitempty"`
	// path to readme
	Readme datastore.Key `json:"readme,omitempty"`
	// Author
	Author    *User       `json:"author,omitempty"`
	Citations []*Citation `json:"citations"`
	Image     string      `json:"image,omitempty"`
	// Description follows the DCAT sense of the word, it should be around a paragraph of human-readable
	// text that outlines the
	Description string `json:"description,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
	IconImage   string `json:"iconImage,omitempty"`
	// Identifier is for *other* data catalog specifications. Identifier should not be used
	// or relied on to be unique, because this package does not enforce any of these rules.
	Identifier string `json:"identifier,omitempty"`
	// License will automatically parse to & from a string value if provided as a raw string
	License *License `json:"license,omitempty"`
	// SemVersion this dataset?
	Version VersionNumber `json:"version,omitempty"`
	// String of Keywords
	Keywords []string `json:"keywords,omitempty"`
	// Contribute
	Contributors []*User `json:"contributors,omitempty"`
	// Languages this dataset is written in
	Language []string `json:"language,omitempty"`
	// Theme
	Theme []*Theme `json:"theme,omitempty"`

	// QueryString is the user-inputted string of this query
	QueryString string `json:"queryString,omitempty"`
	// Query is a path to a query that generated this resource
	Query *Query `json:"query,omitempty"`
	// Syntax this query was written in
	QuerySyntax string `json:"querySyntax,omitempty"`
	// queryPlatform is an identifier for the operating system that performed the query
	QueryPlatform string `json:"queryPlatform,omitempty"`
	// QueryEngine is an identifier for the application that produced the result
	QueryEngine string `json:"queryEngine,omitempty"`
	// QueryEngineConfig outlines any configuration that would affect the resulting hash
	QueryEngineConfig map[string]interface{} `json:"queryEngineConfig,omitempty`
	// Resources is a map of dataset names to dataset references this query is derived from
	// all tables referred to in the query should be present here
	Resources map[string]*Dataset `json:"resources,omitempty"`
}
```

Dataset is stored separately from prescriptive metadata stored in Resource
structs to maximize overlap of the formal query & resource definitions. A
Dataset must resolve to one and only one entity, specified by a `data` property.
It's structure must be specified by a structure definition. This also creates
space for subjective claims about datasets, and allows metadata to take on a
higher frequency of change in contrast to the underlying definition. In
addition, descriptive metadata can and should be author attributed associating
descriptive claims about a resource with a cyptographic keypair which may
represent a person, group of people, or software. This metadata format is also
subject to massive amounts of change. Design goals should include making this
compatible with the DCAT spec, with the one major exception that hashes are
acceptable in place of urls.

#### func  UnmarshalDataset

```go
func UnmarshalDataset(v interface{}) (*Dataset, error)
```
UnmarshalDataset tries to extract a dataset type from an empty interface. Pairs
nicely with datastore.Get() from github.com/ipfs/go-datastore

#### func (*Dataset) Assign

```go
func (d *Dataset) Assign(datasets ...*Dataset)
```
Assign collapses all properties of a group of datasets onto one. this is
directly inspired by Javascript's Object.assign

#### func (*Dataset) IsEmpty

```go
func (ds *Dataset) IsEmpty() bool
```

#### func (*Dataset) MarshalJSON

```go
func (d *Dataset) MarshalJSON() ([]byte, error)
```
MarshalJSON uses a map to combine meta & standard fields. Marshalling a
map[string]interface{} automatically alpha-sorts the keys.

#### func (*Dataset) Meta

```go
func (d *Dataset) Meta() map[string]interface{}
```
Meta gives access to additional metadata not covered by dataset metadata

#### func (*Dataset) Path

```go
func (ds *Dataset) Path() datastore.Key
```

#### func (*Dataset) UnmarshalJSON

```go
func (d *Dataset) UnmarshalJSON(data []byte) error
```
UnmarshalJSON implements json.Unmarshaller

#### type Field

```go
type Field struct {
	Name         string            `json:"name"`
	Type         datatypes.Type    `json:"type,omitempty"`
	MissingValue interface{}       `json:"missingValue,omitempty"`
	Format       string            `json:"format,omitempty"`
	Constraints  *FieldConstraints `json:"constraints,omitempty"`
	Title        string            `json:"title,omitempty"`
	Description  string            `json:"description,omitempty"`
}
```

Field is a field descriptor

#### func (*Field) Assign

```go
func (f *Field) Assign(fields ...*Field)
```

#### func (Field) MarshalJSON

```go
func (f Field) MarshalJSON() ([]byte, error)
```
MarshalJSON satisfies the json.Marshaler interface

#### func (*Field) UnmarshalJSON

```go
func (f *Field) UnmarshalJSON(data []byte) error
```
UnmarshalJSON satisfies the json.Unmarshaler interface

#### type FieldConstraints

```go
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
```

FieldConstraints is supposed to constrain the field, this is totally unfinished,
unimplemented, and needs lots of work TODO - uh, finish this?

#### type FieldKey

```go
type FieldKey []string
```

FieldKey allows a field key to be either a string or object

#### type ForeignKey

```go
type ForeignKey struct {
	Fields FieldKey `json:"fields"`
}
```

ForeignKey is supposed to be for supporting foreign key references. It's also
totally unfinished. TODO - finish this

#### type FormatConfig

```go
type FormatConfig interface {
	Format() DataFormat
	Map() map[string]interface{}
}
```


#### func  NewCsvOptions

```go
func NewCsvOptions(opts map[string]interface{}) (FormatConfig, error)
```

#### func  NewJsonOptions

```go
func NewJsonOptions(opts map[string]interface{}) (FormatConfig, error)
```

#### func  ParseFormatConfigMap

```go
func ParseFormatConfigMap(f DataFormat, opts map[string]interface{}) (FormatConfig, error)
```

#### type JsonOptions

```go
type JsonOptions struct {
	ObjectEntries bool `json:"objectEntries"`
}
```


#### func (*JsonOptions) Format

```go
func (*JsonOptions) Format() DataFormat
```

#### func (*JsonOptions) Map

```go
func (o *JsonOptions) Map() map[string]interface{}
```

#### type License

```go
type License struct {
	Type string `json:"type"`
	Url  string `json:"url"`
}
```

License represents a legal licensing agreement

#### func (License) MarshalJSON

```go
func (l License) MarshalJSON() ([]byte, error)
```
MarshalJSON satisfies the json.Marshaller interface

#### func (*License) UnmarshalJSON

```go
func (l *License) UnmarshalJSON(data []byte) error
```
UnmarshalJSON satisfies the json.Unmarshaller interface

#### type Query

```go
type Query struct {

	// Syntax is an identifier string for the statement syntax (Eg, "SQL")
	Syntax string `json:"syntax"`
	// Structures is a map of all structures referenced in this query,
	// with alphabetical keys generated by datasets in order of appearance within the query.
	// Keys are _always_ referenced in the form [a-z,aa-zz,aaa-zzz, ...] by order of appearence.
	// The query itself is rewritten to refer to these table names using bind variables
	Structures map[string]*Structure `json:"structures"`
	// Statement is the is parsed & rewritten to a _standard form_ to maximize hash overlap.
	// Writing a query to it's standard form involves making deterministic choices to
	// remove non-semantic whitespace, rewrite semantically-equivalent terms like "&&" and "AND"
	// to a chosen version, et cetera.
	// Greater precision of querying format will increase the chances of hash discovery.
	Statement string `json:"statement"`
	// Structure is a path to an algebraic structure that is the _output_ of this structure
	Structure *Structure
}
```

Query defines an action to be taken on one or more structures

#### func  NewQueryRef

```go
func NewQueryRef(path datastore.Key) *Query
```
NewQueryReference creates an empty struct with it's internal path set

#### func  UnmarshalQuery

```go
func UnmarshalQuery(v interface{}) (*Query, error)
```
UnmarshalResource tries to extract a resource type from an empty interface.
Pairs nicely with datastore.Get() from github.com/ipfs/go-datastore

#### func (*Query) IsEmpty

```go
func (q *Query) IsEmpty() bool
```

#### func (Query) MarshalJSON

```go
func (q Query) MarshalJSON() ([]byte, error)
```
MarshalJSON satisfies the json.Marshaler interface

#### func (*Query) Path

```go
func (q *Query) Path() datastore.Key
```

#### func (*Query) UnmarshalJSON

```go
func (q *Query) UnmarshalJSON(data []byte) error
```
UnmarshalJSON satisfies the json.Unmarshaler interface

#### type Schema

```go
type Schema struct {
	Fields     []*Field `json:"fields,omitempty"`
	PrimaryKey FieldKey `json:"primaryKey,omitempty"`
}
```

Schema is analogous to a SQL schema definition

#### func (*Schema) Assign

```go
func (s *Schema) Assign(schemas ...*Schema)
```
AssignSchema collapses all properties of a group of schemas on to one this is
directly inspired by Javascript's Object.assign

#### func (*Schema) FieldForName

```go
func (s *Schema) FieldForName(name string) *Field
```
FieldForName returns the field who's string name matches name

#### func (*Schema) FieldNames

```go
func (s *Schema) FieldNames() (names []string)
```
FieldNames gives a slice of field names defined in schema

#### func (*Schema) FieldTypeStrings

```go
func (s *Schema) FieldTypeStrings() (types []string)
```
FieldTypeStrings gives a slice of each field's type as a string

#### type Structure

```go
type Structure struct {

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
```

Structure designates a deterministic definition for working with a discrete
dataset. Structure is a concrete handle that provides precise details about how
to interpret a given piece of data (the reference to the data itself is provided
elsewhere, specifically in the dataset struct ) These techniques provide
mechanisms for joining & traversing multiple structures. This example is shown
in a human-readable form, for storage on the network the actual output would be
in a condensed, non-indented form, with keys sorted by lexographic order.

#### func  NewStructureRef

```go
func NewStructureRef(path datastore.Key) *Structure
```
NewStructureRef creates an empty struct with it's internal path set

#### func  UnmarshalStructure

```go
func UnmarshalStructure(v interface{}) (*Structure, error)
```
UnmarshalStructure tries to extract a structure type from an empty interface.
Pairs nicely with datastore.Get() from github.com/ipfs/go-datastore

#### func (*Structure) Abstract

```go
func (s *Structure) Abstract() *Structure
```
Abstract returns this structure instance in it's "Abstract" form stripping all
nonessential values & renaming all schema field names to standard variable names

#### func (*Structure) Assign

```go
func (s *Structure) Assign(structures ...*Structure)
```
Assign collapses all properties of a group of structures on to one this is
directly inspired by Javascript's Object.assign

#### func (*Structure) Hash

```go
func (r *Structure) Hash() (string, error)
```
Hash gives the hash of this structure

#### func (*Structure) IsEmpty

```go
func (st *Structure) IsEmpty() bool
```

#### func (Structure) MarshalJSON

```go
func (s Structure) MarshalJSON() (data []byte, err error)
```
MarshalJSON satisfies the json.Marshaler interface

#### func (*Structure) Path

```go
func (s *Structure) Path() datastore.Key
```

#### func (*Structure) StringFieldIndex

```go
func (st *Structure) StringFieldIndex(s string) int
```
StringFieldIndex gives the index of a field who's name matches s it returns -1
if no match is found

#### func (*Structure) UnmarshalJSON

```go
func (s *Structure) UnmarshalJSON(data []byte) error
```
UnmarshalJSON satisfies the json.Unmarshaler interface

#### func (*Structure) Valid

```go
func (ds *Structure) Valid() error
```
Valid validates weather or not this structure

#### type Theme

```go
type Theme struct {
	Description     string `json:"description,omitempty"`
	DisplayName     string `json:"display_name,omitempty"`
	ImageDisplayUrl string `json:"image_display_url,omitempty"`
	Id              string `json:"id,omitempty"`
	Name            string `json:"name,omitempty"`
	Title           string `json:"title,omitempty"`
}
```

Theme is pulled from the Project Open Data Schema version 1.1

#### type User

```go
type User struct {
	Fullname string `fn,omitempty`
	Email    string `email,omitempty`
}
```

User is a placholder for talking about people, groups, organizations

#### type VariableName

```go
type VariableName string
```

VariableName is a string that conforms to standard variable naming conventions
must start with a letter, no spaces TODO - we're not really using this much,
consider depricating, or using properly

#### func (VariableName) MarshalJSON

```go
func (name VariableName) MarshalJSON() ([]byte, error)
```
MarshalJSON satisfies the json.Marshaller interface

#### func (*VariableName) UnmarshalJSON

```go
func (name *VariableName) UnmarshalJSON(data []byte) error
```
UnmarshalJSON satisfies the json.Unmarshaller interface

#### type VersionNumber

```go
type VersionNumber string
```

VersionNumber is a semantic major.minor.patch TODO - make Version enforce this
format
