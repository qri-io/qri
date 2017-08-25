// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
				cmd.PrintErr(err)
			} else {
				fmt.Printf("%v\n", r)
			}
		}
	}()

	cmd.Execute()
}
