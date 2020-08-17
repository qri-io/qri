package cmd

import (
	"testing"
)

func TestDoubleCheckout(t *testing.T) {
	runner := NewFSITestRunner(t, "test_peer_double_checkout", "double_checkout")
	defer runner.Delete()

	_ = runner.CreateAndChdirToWorkDir("checkout_dir")

	// Init as a linked directory
	if err := runner.ExecCommand("qri init --name double_checkout --format csv"); err != nil {
		t.Fatal(err.Error())
	}

	// Save a version of the dataset
	if err := runner.ExecCommand("qri save"); err != nil {
		t.Fatal(err.Error())
	}

	// Move to different work directory
	runner.ChdirToRoot()

	// Checkout should fail as the dataset already linked
	if err := runner.ExecCommand("qri checkout me/double_checkout ./checkout_test"); err == nil {
		t.Fatal("`qri checkout` should fail if already linked, but did not fail")
	}
}
