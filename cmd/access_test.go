package cmd

import (
	"testing"
)

func TestAccessCreateToken(t *testing.T) {
	run := NewTestRunner(t, "peer", "cmd_test_create_access_token")
	defer run.Delete()

	run.MustExecCombinedOutErr(t, "qri access token --for me")
	run.MustExecCombinedOutErr(t, "qri access token --for peer")
}
