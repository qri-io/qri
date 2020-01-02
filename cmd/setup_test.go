package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/ioes"
)

func TestSetupComplete(t *testing.T) {
	streams, _, _, _ := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	opt := &SetupOptions{
		IOStreams: streams,
	}

	opt.Complete(f, nil)
}

func TestSetupGimmeDoggo(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "")

	actual := run.MustExec(t, "qri setup --gimme-doggo")
	expect := "testnick\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}
