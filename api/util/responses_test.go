package util

import (
	"net/http/httptest"
	"testing"
)

func TestWriteResponseWithNextPageJSON(t *testing.T) {
	rr := httptest.NewRecorder()

	if err := WriteResponseWithNextPageJSON(rr, "data", "/next", `{"start":5}`); err != nil {
		t.Fatal(err)
	}

	actual := rr.Body.String()
	expect := `{"data":"data","meta":{"code":200},"nextPage":{"method":"POST","url":"/next","contentType":"application/json","jsonBody":"{\"start\":5}"}}`

	if expect != actual {
		t.Errorf("result mismatch. expected:\n%s\ngot:\n%s", expect, actual)
	}
}
