package cdxj

import (
	"bytes"
	"testing"
)

func TestWriter(t *testing.T) {
	buf := &bytes.Buffer{}

	w := NewWriter(buf)
	for i, rec := range parsed {
		if err := w.Write(rec); err != nil {
			t.Errorf("error writing record %d: %s", i, err.Error())
			return
		}
	}

	if err := w.Close(); err != nil {
		t.Errorf("close error: %s", err.Error())
		return
	}

	if buf.String() != eg {
		t.Errorf("result mismatch expected:\n%s\ngot:\n%s", eg, buf.String())
	}
}
