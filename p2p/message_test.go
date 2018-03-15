package p2p

import (
	"bytes"
	"testing"
	"time"
)

func TestMessageUpdate(t *testing.T) {
	a := Message{
		ID:       "a",
		Created:  time.Now(),
		Deadline: time.Now().Add(time.Minute),
		Body:     []byte("foo"),
	}

	b := a.Update([]byte("bar"))

	if !bytes.Equal(b.Body, []byte("bar")) {
		t.Errorf("payload mismatch. expected %s, got: %s", "bar", string(b.Body))
	}
}
