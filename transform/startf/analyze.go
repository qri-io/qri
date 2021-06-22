package startf

import (
	"fmt"
	"go.starlark.net/starlark"
	starlarkSyntax "go.starlark.net/syntax"
)

func analyzeScriptFile(thread *starlark.Thread, filename string) {
	// ExecFile(thread *Thread, filename string, src interface{}, predeclared StringDict)
	// SourceProgram(filename string, src interface{}, isPredeclared func(string) bool)
	// f, err := syntax.Parse(filename string, src interface{}, 0 ?)
	fmt.Printf("analyze: %s\n", filename)

	f, err := starlarkSyntax.Parse(filename, nil, 0)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Parsed successfully!\n")
	_ = f
}
