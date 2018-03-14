package main

import (
	"fmt"

	"github.com/qri-io/qri/cmd"
)

func main() {
	// Catch errors & pretty-print.
	// comment this out to get stack traces back.
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				fmt.Println(err.Error())
			} else {
				fmt.Println(r)
			}
		}
	}()

	cmd.Execute()
}
