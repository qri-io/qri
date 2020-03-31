package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSetupComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_setup_complete")
	defer run.Delete()

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	opt := &SetupOptions{
		IOStreams: run.Streams,
	}

	opt.Complete(f, nil)
}

func TestSetupGimmeDoggo(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_gimme_doggo")
	defer run.Delete()

	actual := run.MustExec(t, "qri setup --gimme-doggo")
	expect := "testnick\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}
