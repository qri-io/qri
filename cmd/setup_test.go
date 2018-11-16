package cmd

import (
	"testing"

	"github.com/qri-io/ioes"
)

func TestSetupComplete(t *testing.T) {
	streams, _, _, _ := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory(nil)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	opt := &SetupOptions{
		IOStreams: streams,
	}

	opt.Complete(f, nil)
}
