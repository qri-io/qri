package mock

import (
	"testing"
)

func TestMockServer(t *testing.T) {
	NewMockServer()
	NewMockServerRegistry(NewMemRegistry(nil))
}
