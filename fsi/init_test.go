package fsi

import (
	"bytes"
	"fmt"
	"testing"
)

func TestCreateBasicStructure(t *testing.T) {
	good := []struct {
		format      string
		schema      map[string]interface{}
		expectBytes []byte
	}{
		{"csv", nil, []byte(`{"format":"csv","formatConfig":{"headerRow":false,"lazyQuotes":false,"variadicFields":false},"schema":null}`)},
		{"json", nil, []byte(`{"format":"json","formatConfig":{"pretty":false},"schema":null}`)},
		{"xlsx", nil, []byte(`{"format":"xlsx","formatConfig":{"sheetName":"sheet1"},"schema":null}`)},
		{"csv", map[string]interface{}{"items": map[string]interface{}{"items": "array"}}, []byte(`{"format":"csv","formatConfig":{"headerRow":false,"lazyQuotes":false,"variadicFields":false},"schema":{"items":{"items":"array"}}}`)},
		{"csv", map[string]interface{}{}, []byte(`{"format":"csv","formatConfig":{"headerRow":false,"lazyQuotes":false,"variadicFields":false},"schema":null}`)},
	}
	for _, c := range good {
		t.Run(fmt.Sprintf("good: %s", c.format), func(t *testing.T) {
			gotBytes, err := createBasicStructure(c.format, c.schema)
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
			_, err := createBasicStructure(c.format, nil)
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
