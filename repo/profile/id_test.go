package profile

import (
	"bytes"
	"testing"
)

func TestIDJSON(t *testing.T) {
	idbytes, err := IDB58MustDecode("QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1").MarshalJSON()
	if err != nil {
		t.Error(err.Error())
		return
	}
	expect := []byte(`"QmRdexT18WuAKVX3vPusqmJTWLeNSeJgjmMbaF5QLGHna1"`)
	if !bytes.Equal(idbytes, expect) {
		t.Errorf("byte mistmatch. expected: %s, got: %s", string(expect), string(idbytes))
	}
}
