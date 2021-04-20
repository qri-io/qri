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
	"reflect"
	"strings"
	"text/template"

	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/version"
)

type Docs struct {
	QriVersion   string
	LibMethods   []LibMethod
	ParamStructs []LibMethodParams
}

type LibMethod struct {
	MethodSet  string
	MethodName string
	Doc        string
	Params     LibMethodParams
	Endpoint   lib.APIEndpoint
	HTTPVerb   string
}

type LibMethodParams struct {
	Name   string
	Doc    string
	Fields []LibMethodField
}

type LibMethodField struct {
	Name    string
	Type    string
	Doc     string
	Tags    string
	Comment string
}

func OpenAPIYAML() *bytes.Buffer {
	params := parseLibMethodParams()
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
				Params:     params[inType.Name()],
			}
			methods = append(methods, m)
		}
	}

	paramSlice := make([]LibMethodParams, 0, len(params))
	for _, param := range params {
		paramSlice = append(paramSlice, param)
	}

	d := Docs{
		QriVersion:   version.Version,
		LibMethods:   methods,
		ParamStructs: paramSlice,
	}

	tmpl := template.Must(template.ParseFiles("api_doc_template.yaml"))
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, d); err != nil {
		panic(err)
	}

	return buf
}

func parseLibMethodParams() map[string]LibMethodParams {
	params := map[string]LibMethodParams{}
	// Create the AST by parsing src and test.
	fset := token.NewFileSet()

	fileNames := []string{
		"/Users/b5/qri/qri/lib/datasets.go",
	}

	fileStrings := []string{}
	files := []*ast.File{}

	for i, filename := range fileNames {
		fileData, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		fileStrings = append(fileStrings, string(fileData))
		files = append(files, mustParse(fset, filename, fileStrings[i]))
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
					fields := make([]LibMethodField, 0, len(structSpec.Fields.List))
					for _, f := range structSpec.Fields.List {
						field := LibMethodField{
							Name:    f.Names[0].String(),
							Type:    typeToString(fset, f.Type),
							Comment: f.Comment.Text(),
						}
						if f.Doc != nil {
							field.Doc = sanitizeDocString(f.Doc.Text())
						}
						if f.Tag != nil {
							field.Tags = f.Tag.Value
						}
						fields = append(fields, field)
					}

					p := LibMethodParams{
						Name:   typeSpec.Name.String(),
						Doc:    sanitizeDocString(typeSpec.Comment.Text()),
						Fields: fields,
					}
					params[typeSpec.Name.String()] = p
				}
			}
		}
	}

	return params
}

func mustParse(fset *token.FileSet, filename, file string) *ast.File {
	f, err := parser.ParseFile(fset, filename, file, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	return f
}

func sanitizeDocString(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\"", "'")
	return s
}

func typeToString(fset *token.FileSet, exp ast.Expr) string {
	buf := &bytes.Buffer{}
	printer.Fprint(buf, fset, exp)
	str := buf.String()
	typeMap := map[string]string{
		"*dataset.Dataset":     "Dataset",
		"*dag.Manifest":        "Manifest",
		"*dsref.Rev":           "Revision",
		"[]byte":               "Bytes",
		"[]string":             "String Array",
		"map[string]string":    "Record",
		"io.Writer":            "Writer",
		"dataset.FormatConfig": "FormatConfig",

		// map to jsonschema types:
		"bool":    "boolean",
		"int":     "number",
		"float32": "number",
		"float64": "number",
	}

	if replace, ok := typeMap[str]; ok {
		str = replace
	}
	return str
}
