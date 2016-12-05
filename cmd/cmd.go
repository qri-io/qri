package cmd

import (
	"fmt"
	"os"
)

// ErrExit writes an error to stdout & exits
func ErrExit(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}
