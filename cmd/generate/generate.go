// package generates generates a bash completion file for qri
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	lastArg := os.Args[len(os.Args)-1]

	switch lastArg {
	case "completions":
		fmt.Printf("generating completions file...")
		root := cmd.NewQriCommand(cmd.EnvPathFactory, ioes.NewStdIOStreams())
		root.GenBashCompletionFile("out.sh")
		fmt.Println("done")
	case "docs":
		fmt.Printf("generating markdown docs...")
		path := "docs"
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			log.Fatal(err)
		}
		root := cmd.NewQriCommand(cmd.EnvPathFactory, ioes.NewStdIOStreams())
		err := doc.GenMarkdownTree(root, path)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("done")
	default:
		fmt.Println("please provide a generate argument: [docs|completions]")
	}
}
