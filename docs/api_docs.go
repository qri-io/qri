package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/version"
)

type docs struct {
	QriVersion string
	LibMethods []libMethod
	Types      []qriType
}

type libMethod struct {
	MethodSet  string
	MethodName string
	Doc        string
	Params     qriType
	Endpoint   lib.APIEndpoint
	HTTPVerb   string
	Response   response
	Paginated  bool
}

type qriType struct {
	Name        string
	Doc         string
	Fields      []field
	WriteToSpec bool
}

type field struct {
	Name         string
	Type         string
	TypeIsCommon bool
	Doc          string
	Tags         string
	Comment      string
}

type response struct {
	Type    string
	IsArray bool
}

// OpenAPIYAML generates the OpenAPI Spec for the Qri API
func OpenAPIYAML() (*bytes.Buffer, error) {
	qriTypes, err := parseQriTypes()
	if err != nil {
		return nil, err
	}

	var methods []libMethod

	var nilInst *lib.Instance
	for _, methodSet := range nilInst.AllMethods() {
		msetType := reflect.TypeOf(methodSet)
		methodAttributes := methodSet.Attributes()

		// Iterate methods on the implementation, register those that have the right signature
		num := msetType.NumMethod()
		for k := 0; k < num; k++ {
			i := msetType.Method(k)
			f := i.Type
			if f.NumIn() != 3 {
				continue
			}

			// Second input (after receiver) is a pointer to the input struct
			inType := f.In(2)
			if inType.Kind() != reflect.Ptr {
				fmt.Printf("%q input 1 is not a pointer. got %s\n", i.Name, inType.Kind())
				continue
			}
			inType = inType.Elem()
			if inType.Kind() != reflect.Struct {
				fmt.Printf("%q input 1 is not a pointer to a struct. got %s\n", i.Name, inType.Kind())
				continue
			}

			attrs := methodAttributes[strings.ToLower(i.Name)]
			if attrs.Endpoint == lib.DenyHTTP {
				continue
			}

			// Validate the output values of the implementation
			numOuts := f.NumOut()
			if numOuts < 1 || numOuts > 3 {
				fmt.Printf("%s: bad number of outputs: %d\n", i.Name, numOuts)
				continue
			}
			// Validate output values
			var outType reflect.Type
			outTypeName := ""
			outIsArray := false
			returnsCursor := false
			if numOuts == 2 || numOuts == 3 {
				// First output is anything
				outType = f.Out(0)
			}

			if outType == nil {
				outTypeName = "nil"
			} else {
				if outType.Kind() == reflect.Ptr {
					outType = outType.Elem()
				}
				if outType.Kind() == reflect.Slice {
					outType = outType.Elem()
					outIsArray = true
				}
				outTypeName = outType.String()

				// all lib structs are already defined
				outTypeName = strings.TrimPrefix(outTypeName, "lib.")
			}

			outTypeName = getMappedType(outTypeName)

			if outTypeName == "string" || outTypeName == "Bytes" {
				outTypeName = "RawResponse"
			}

			if numOuts == 3 {
				// Second output must be a cursor
				outCursorType := f.Out(1)
				if outCursorType.Name() != "Cursor" {
					fmt.Printf("%s: second output val must be a cursor, got %v\n", i.Name, outCursorType)
					// continue
				}
				returnsCursor = true
			}

			// TODO(arqu): we can use error types for the response definitions as well
			// for now a generic one will do
			// Last output must be an error
			outErrType := f.Out(numOuts - 1)
			if outErrType.Name() != "error" {
				fmt.Printf("%s: last output val should be error, got %v\n", i.Name, outErrType)
				// continue
			}

			if !isDefinedResponse(outTypeName, qriTypes) {
				fmt.Printf("%s: '%s' output response type not defined (%s)\n", i.Name, outTypeName, inType.Name())
				outTypeName = "NotDefined"
			}

			if qt, ok := qriTypes[outTypeName]; ok {
				qt.WriteToSpec = true
				qriTypes[outTypeName] = qt
			}

			if returnsCursor {
				fmt.Printf("%s: is paginated", i.Name)
			}

			m := libMethod{
				MethodSet:  methodSet.Name(),
				MethodName: i.Name,
				Endpoint:   attrs.Endpoint,
				HTTPVerb:   strings.ToLower(attrs.HTTPVerb),
				Params:     qriTypes[inType.Name()],
				Paginated:  returnsCursor,
				Response: response{
					Type:    outTypeName,
					IsArray: outIsArray,
				},
			}
			methods = append(methods, m)
		}
	}

	qriTypeSlice := make([]qriType, 0, len(qriTypes))
	for _, qriType := range qriTypes {
		if qriType.WriteToSpec {
			qriTypeSlice = append(qriTypeSlice, qriType)
		}
	}

	d := docs{
		QriVersion: version.Version,
		LibMethods: methods,
		Types:      qriTypeSlice,
	}

	tmpl := template.Must(template.ParseFiles("api_doc_template.yaml"))
	buf := &bytes.Buffer{}

	err = tmpl.Execute(buf, d)

	buf = sanitizeOutput(buf)
	return buf, err
}

func sanitizeOutput(buf *bytes.Buffer) *bytes.Buffer {
	s := buf.String()
	s = strings.Replace(s, "\n\n", "\n", -1)
	res := &bytes.Buffer{}
	res.WriteString(s)
	return res
}

func isDefinedResponse(r string, qriTypes map[string]qriType) bool {
	responseMap := map[string]bool{
		// Placeholders
		"Dataset":                   true,
		"VersionInfo":               true,
		"StatusItem":                true,
		"Profile":                   true,
		"Ref":                       true,
		"DAGManifest":               true,
		"DAGInfo":                   true,
		"ChangeReport":              true,
		"MappedArraysOfVersionInfo": true,

		// Implemented
		"RawResponse": true,
		"Nil":         true,
		"NotDefined":  true,
	}

	if res, ok := responseMap[r]; ok {
		return res
	}

	if qriTypes != nil {
		_, isQriType := qriTypes[r]
		return isQriType
	}
	return false
}

func parseQriTypes() (map[string]qriType, error) {
	params := map[string]qriType{}
	// Create the AST by parsing src and test.
	fset := token.NewFileSet()

	libFiles, err := ioutil.ReadDir("../lib/")
	if err != nil {
		return nil, err
	}

	files := []*ast.File{}

	for _, fInfo := range libFiles {
		if !strings.HasSuffix(fInfo.Name(), ".go") {
			continue
		}

		path := filepath.Join("../lib/", fInfo.Name())
		astFile, err := readASTFile(fset, path)
		if err != nil {
			return nil, fmt.Errorf("reading AST from go file %q %w: ", path, err)
		}
		files = append(files, astFile)
	}

	// Compute package documentation
	p, err := doc.NewFromFiles(fset, files, "github.com/qri-io/qri/lib", doc.PreserveAST)
	if err != nil {
		panic(err)
	}

	for _, t := range p.Types {
		for _, spec := range t.Decl.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok {
				if structSpec, ok := typeSpec.Type.(*ast.StructType); ok {
					fields := make([]field, 0, len(structSpec.Fields.List))
					for _, f := range structSpec.Fields.List {
						if len(f.Names) == 0 {
							continue
						}

						t, common := typeToString(fset, f.Type)
						field := field{
							Name:         f.Names[0].String(),
							Type:         t,
							TypeIsCommon: common,
							Comment:      f.Comment.Text(),
						}
						if f.Doc != nil {
							field.Doc = sanitizeDocString(f.Doc.Text())
						}
						if f.Tag != nil {
							field.Tags = f.Tag.Value
						}
						fields = append(fields, field)
					}

					writeToSpec := strings.HasSuffix(typeSpec.Name.String(), "Params") || strings.HasSuffix(typeSpec.Name.String(), "ParamsPod")
					p := qriType{
						Name:        typeSpec.Name.String(),
						Doc:         sanitizeDocString(typeSpec.Comment.Text()),
						Fields:      fields,
						WriteToSpec: writeToSpec,
					}
					params[typeSpec.Name.String()] = p
				}
			}
		}
	}

	return params, nil
}

func readASTFile(fset *token.FileSet, filepath string) (*ast.File, error) {
	fileData, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return parser.ParseFile(fset, filepath, string(fileData), parser.ParseComments)
}

func sanitizeDocString(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\"", "'")
	return s
}

func getMappedType(f string) string {

	f = strings.TrimPrefix(f, "*")

	typeMap := map[string]string{
		// TODO(b5): we should get these data types captured in the type map, but many
		// aren't defined in lib, or don't end in "Params". Lots of these are used
		// as repsonse objects
		"dataset.Dataset":                "Dataset",
		"dataset.Structure":              "DatasetStructure",
		"dataset.Transform":              "Transform",
		"dag.Manifest":                   "DAGManifest",
		"dag.Info":                       "DAGInfo",
		"dsref.Rev":                      "Revision",
		"dsref.Ref":                      "Ref",
		"dsref.VersionInfo":              "VersionInfo",
		"[]uint8":                        "Bytes",
		"uint8":                          "Bytes",
		"[]byte":                         "Bytes",
		"[]string":                       "StringArray",
		"map[string]string":              "Record",
		"io.Writer":                      "Writer",
		"dataset.FormatConfig":           "FormatConfig",
		"config.ProfilePod":              "Profile",
		"config.Config":                  "Config",
		"key.CryptoGenerator":            "CryptoGenerator",
		"profile.ID":                     "ProfileID",
		"RegistryProfile":                "RegistryProfile",
		"nil":                            "Nil",
		"fsi.StatusItem":                 "StatusItem",
		"changes.ChangeReportResponse":   "ChangeReport",
		"map[string][]dsref.VersionInfo": "MappedArraysOfVersionInfo",
		"[]*Delta":                       "DeltaValues",
		"json.RawMessage":                "Bytes",
		"ioes.IOStreams":                 "Nil",
		"[]jsonschema.KeyError":          "JSONKeyErrors",
		"time.Duration":                  "DurationString",

		// map go types to jsonschema types:
		"bool":    "boolean",
		"int":     "number",
		"float32": "number",
		"float64": "number",
	}

	if replace, ok := typeMap[f]; ok {
		f = replace
	}

	return f
}

func typeToString(fset *token.FileSet, exp ast.Expr) (typ string, isJSONSchemaType bool) {
	buf := &bytes.Buffer{}
	printer.Fprint(buf, fset, exp)
	str := buf.String()

	str = getMappedType(str)

	_, isJSONSchemaType = map[string]struct{}{
		"array":   struct{}{},
		"boolean": struct{}{},
		"integer": struct{}{},
		"number":  struct{}{},
		"object":  struct{}{},
		"string":  struct{}{},
	}[str]

	return str, isJSONSchemaType
}
