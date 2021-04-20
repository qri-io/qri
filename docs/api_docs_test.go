package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIYAML(t *testing.T) {
	buf, err := OpenAPIYAML()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(buf.String())

	sw, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := sw.Validate(context.Background()); err != nil {
		t.Error(err)
	}
}
