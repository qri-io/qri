package lib

import (
	"fmt"
	"testing"
)

func TestError(t *testing.T) {
	e := NewError(fmt.Errorf("testing error"), "testing message")

	if e.Message() != "testing message" {
		t.Errorf("error in Error struct function `Message()`: expected: %s, got: %s", "testing message", e.Message())
	}

	if e.Error() != "testing error" {
		t.Errorf("error in Error struct function `Error()`: expected: %s, got: %s", "testing error", e.Error())
	}
}
