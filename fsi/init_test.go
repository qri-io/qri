package fsi

import (
	"bytes"
	"fmt"
	"testing"
)

func TestCreateBasicStructure(t *testing.T) {
	good := []struct {
		format      string
		expectBytes []byte
	}{
		{"csv", []byte(`{"format":"csv","formatConfig":{"headerRow":false,"lazyQuotes":false,"variadicFields":false}}`)},
		{"json", []byte(`{"format":"json","formatConfig":{"pretty":false}}`)},
		{"xlsx", []byte(`{"format":"xlsx","formatConfig":{"sheetName":"sheet1"}}`)},
	}
	for _, c := range good {
		t.Run(fmt.Sprintf("good: %s", c.format), func(t *testing.T) {
			gotBytes, err := createBasicStructure(c.format)
			if err != nil {
				t.Errorf("expected no error. got: %s", err)
			}
			if !bytes.Equal(gotBytes, c.expectBytes) {
				t.Errorf("expected '%s', got '%s'", c.expectBytes, gotBytes)
			}
		})
	}
	bad := []struct {
		format string
		err    string
	}{
		{"bad_format", "unknown body format 'bad_format'"},
	}
	for _, c := range bad {
		t.Run(fmt.Sprintf("bad: %s", c.format), func(t *testing.T) {
			_, err := createBasicStructure(c.format)
			t.Log(err)
			if err == nil {
				t.Errorf("expected error. got: %s", err)
			}
			if err.Error() != c.err {
				t.Errorf("expected error '%s'. got: '%s'", c.err, err)
			}
		})
	}
}
