package main

import (
	"fmt"
	"testing"
)

func TestOpenAPIYAML(t *testing.T) {
	buf := OpenAPIYAML()
	fmt.Println(buf.String())
}
