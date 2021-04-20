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

type Docs struct {
	QriVersion string
	LibMethods []LibMethod
	Types      []QriType
}

type LibMethod struct {
	MethodSet  string
	MethodName string
	Doc        string
	Params     QriType
	Endpoint   lib.APIEndpoint
	HTTPVerb   string
}

type QriType struct {
	Name   string
	Doc    string
	Fields []Field
}

type Field struct {
	Name         string
	Type         string
	TypeIsCommon bool
	Doc          string
	Tags         string
	Comment      string
}

func OpenAPIYAML() (*bytes.Buffer, error) {
	qriTypes, err := parseQriTypes()
	if err != nil {
		return nil, err
	}

	var methods []LibMethod

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
				// fmt.Printf("%q does not have 2 inputs. Instead has %d\n", i.Name, f.NumIn())
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
			m := LibMethod{
				MethodSet:  methodSet.Name(),
				MethodName: i.Name,
				Endpoint:   attrs.Endpoint,
				HTTPVerb:   strings.ToLower(attrs.HTTPVerb),
				Params:     qriTypes[inType.Name()],
			}
			methods = append(methods, m)
		}
	}

	qriTypeSlice := make([]QriType, 0, len(qriTypes))
	for _, qriType := range qriTypes {
		qriTypeSlice = append(qriTypeSlice, qriType)
	}

	d := Docs{
		QriVersion: version.Version,
		LibMethods: methods,
		Types:      qriTypeSlice,
	}

	tmpl := template.Must(template.ParseFiles("api_doc_template.yaml"))
	buf := &bytes.Buffer{}

	err = tmpl.Execute(buf, d)
	return buf, err
}

func parseQriTypes() (map[string]QriType, error) {
	params := map[string]QriType{}
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

	p.Filter(func(name string) bool { return strings.HasSuffix(name, "Params") })

	for _, t := range p.Types {
		for _, spec := range t.Decl.Specs {
			if typeSpec, ok := spec.(*ast.TypeSpec); ok {
				if structSpec, ok := typeSpec.Type.(*ast.StructType); ok {
					fields := make([]Field, 0, len(structSpec.Fields.List))
					for _, f := range structSpec.Fields.List {
						if len(f.Names) == 0 {
							// fmt.Printf("skipping unnamed (probably embedded struct) field in %q\n", typeSpec.Name)
							continue
						}

						t, common := typeToString(fset, f.Type)
						field := Field{
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

					p := QriType{
						Name:   typeSpec.Name.String(),
						Doc:    sanitizeDocString(typeSpec.Comment.Text()),
						Fields: fields,
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

func typeToString(fset *token.FileSet, exp ast.Expr) (typ string, isJSONSchemaType bool) {
	buf := &bytes.Buffer{}
	printer.Fprint(buf, fset, exp)
	str := buf.String()
	typeMap := map[string]string{
		// TODO(b5): we should get these data types captured in the type map, but many
		// aren't defined in lib, or don't end in "Params". Lots of these are used
		// as repsonse objects
		"*dataset.Dataset":     "Dataset",
		"*dag.Manifest":        "Manifest",
		"*dsref.Rev":           "Revision",
		"[]byte":               "Bytes",
		"[]string":             "String Array",
		"map[string]string":    "Record",
		"io.Writer":            "Writer",
		"dataset.FormatConfig": "FormatConfig",
		"config.ProfilePod":    "Profile",
		"*config.ProfilePod":   "Profile",
		"*dataset.Transform":   "Transform",
		"*config.Config":       "Config",
		"key.CryptoGenerator":  "CryptoGenerator",
		"profile.ID":           "ProfileID",
		"*RegistryProfile":     "RegistryProfile",

		// map go types to jsonschema types:
		"bool":    "boolean",
		"int":     "number",
		"float32": "number",
		"float64": "number",
	}

	if replace, ok := typeMap[str]; ok {
		str = replace
	}

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
