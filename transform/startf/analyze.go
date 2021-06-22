package startf

import (
	"encoding/json"
	"fmt"
	"go.starlark.net/starlark"
	starlarkSyntax "go.starlark.net/syntax"
)

func analyzeScriptFile(thread *starlark.Thread, filename string) {
	err := doAnalyze(thread, filename)
	if err != nil {
		panic(err)
	}
}

func doAnalyze(thread *starlark.Thread, filename string) error {
	// ExecFile(thread *Thread, filename string, src interface{}, predeclared StringDict)
	// SourceProgram(filename string, src interface{}, isPredeclared func(string) bool)
	// f, err := syntax.Parse(filename string, src interface{}, 0 ?)
	fmt.Printf("analyze: %s\n", filename)

	f, err := starlarkSyntax.Parse(filename, nil, 0)
	if err != nil {
		return err
	}

	fmt.Printf("Parsed successfully!\n")
	data, err := json.MarshalIndent(f, "", " ")
	if err != nil {
		return err
	}

	text := string(data)
	fmt.Printf("%s\n", text)

	return nil
}
