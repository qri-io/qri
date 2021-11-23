package util

import (
	"net/http/httptest"
	"testing"
)

func TestWriteResponseWithNextPage(t *testing.T) {
	rr := httptest.NewRecorder()

	if err := WriteResponseWithNextPage(rr, "data", "/next", map[string]string{"start":"5"}); err != nil {
		t.Fatal(err)
	}

	actual := rr.Body.String()
	expect := `{"data":"data","meta":{"code":200},"nextPage":{"url":"/next","params":{"start":"5"}}}`
	if expect != actual {
		t.Errorf("result mismatch. expected:\n%s\ngot:\n%s", expect, actual)
	}
}
