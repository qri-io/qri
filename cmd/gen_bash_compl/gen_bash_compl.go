// package gen_bash_compl generates a bash completion file for qri
package main

import (
	"github.com/qri-io/qri/cmd"
)

func main() {
	cmd.RootCmd.GenBashCompletionFile("out.sh")
}
