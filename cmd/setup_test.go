package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/auth/key"
	test_peers "github.com/qri-io/qri/config/test"
	repotest "github.com/qri-io/qri/repo/test"
	"github.com/spf13/cobra"
)

// Test that setup command object can be created
func TestSetupComplete(t *testing.T) {
	run := NewTestRunner(t, "test_peer_setup_complete", "qri_test_setup_complete")
	defer run.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := NewTestFactory(ctx)
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	opt := &SetupOptions{
		IOStreams: run.Streams,
	}

	opt.Complete(f, nil)
}

// Test that setup run with --gimme-doggo command returns a default nickname
func TestSetupGimmeDoggo(t *testing.T) {
	run := NewTestRunner(t, "test_peer_gimme_doggo", "qri_test_gimme_doggo")
	defer run.Delete()

	actual := run.MustExec(t, "qri setup --gimme-doggo")
	expect := "testnick\n"
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}
}

// NOTE: These tests below do not use the TestRunner. This is because the TestRunner creates
// its own MockRepo, which removes the need to run `qri setup`. Since the whole idea here is
// to test `qri setup`'s behavior, we cannot use that functionality.

// Test that setup with no input will use the suggested username
func TestSetupWithNoInput(t *testing.T) {
	ctx := context.Background()
	info := test_peers.GetTestPeerInfo(0)

	qriHome := createTmpQriHome(t)
	cmd, shutdown := newCommand(ctx, qriHome, repotest.NewTestCrypto())

	cmdText := "qri setup"
	if err := executeCommand(cmd, cmdText); err != nil {
		timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
		t.Fatal(err)
	}

	configData := readConfigFile(t, qriHome)

	username := configData["Profile"].(map[string]interface{})["peername"]
	expect := "testnick"
	if username != expect {
		t.Errorf("setup didn't create correct username, expect: %s, got: %s", expect, username)
	}

	profileID := configData["Profile"].(map[string]interface{})["id"]
	if profileID != info.EncodedPeerID {
		t.Errorf("setup didn't create correct profileID, expect: %s, got: %s", info.EncodedPeerID, profileID)
	}

	privkey := configData["Profile"].(map[string]interface{})["privkey"]
	if privkey != info.EncodedPrivKey {
		t.Errorf("setup didn't create correct private key")
	}
}

// Test that setup with a response on stdin will use that username from stdin
func TestSetupUsernameOnStdin(t *testing.T) {
	ctx := context.Background()
	info := test_peers.GetTestPeerInfo(0)

	qriHome := createTmpQriHome(t)
	stdinText := "qri_test_name"
	cmd, shutdown := newCommandWithStdin(ctx, qriHome, stdinText, repotest.NewTestCrypto())

	cmdText := "qri setup"
	if err := executeCommand(cmd, cmdText); err != nil {
		timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
		t.Fatal(err)
	}

	configData := readConfigFile(t, qriHome)

	username := configData["Profile"].(map[string]interface{})["peername"]
	expect := "qri_test_name"
	if username != expect {
		t.Errorf("setup didn't create correct username, expect: %s, got: %s", expect, username)
	}

	profileID := configData["Profile"].(map[string]interface{})["id"]
	if profileID != info.EncodedPeerID {
		t.Errorf("setup didn't create correct profileID, expect: %s, got: %s", info.EncodedPeerID, profileID)
	}

	privkey := configData["Profile"].(map[string]interface{})["privkey"]
	if privkey != info.EncodedPrivKey {
		t.Errorf("setup didn't create correct private key")
	}
}

// Test that setup doesn't prompt if given anonymous flag
func TestSetupAnonymousIgnoresStdin(t *testing.T) {
	ctx := context.Background()
	info := test_peers.GetTestPeerInfo(0)

	qriHome := createTmpQriHome(t)
	stdinText := "qri_test_name"
	cmd, shutdown := newCommandWithStdin(ctx, qriHome, stdinText, repotest.NewTestCrypto())

	cmdText := "qri setup --anonymous"
	if err := executeCommand(cmd, cmdText); err != nil {
		timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
		t.Fatal(err)
	}

	configData := readConfigFile(t, qriHome)

	username := configData["Profile"].(map[string]interface{})["peername"]
	expect := "testnick"
	if username != expect {
		t.Errorf("setup didn't create correct username, expect: %s, got: %s", expect, username)
	}

	profileID := configData["Profile"].(map[string]interface{})["id"]
	if profileID != info.EncodedPeerID {
		t.Errorf("setup didn't create correct profileID, expect: %s, got: %s", info.EncodedPeerID, profileID)
	}

	privkey := configData["Profile"].(map[string]interface{})["privkey"]
	if privkey != info.EncodedPrivKey {
		t.Errorf("setup didn't create correct private key")
	}
}

// Test that setup with the --username flag will use that value
func TestSetupUsernameFlag(t *testing.T) {
	ctx := context.Background()
	info := test_peers.GetTestPeerInfo(0)

	qriHome := createTmpQriHome(t)
	cmd, shutdown := newCommand(ctx, qriHome, repotest.NewTestCrypto())

	cmdText := "qri setup --username qri_cool_user"
	if err := executeCommand(cmd, cmdText); err != nil {
		timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
		t.Fatal(err)
	}

	configData := readConfigFile(t, qriHome)

	username := configData["Profile"].(map[string]interface{})["peername"]
	expect := "qri_cool_user"
	if username != expect {
		t.Errorf("setup didn't create correct username, expect: %s, got: %s", expect, username)
	}

	profileID := configData["Profile"].(map[string]interface{})["id"]
	if profileID != info.EncodedPeerID {
		t.Errorf("setup didn't create correct profileID, expect: %s, got: %s", info.EncodedPeerID, profileID)
	}

	privkey := configData["Profile"].(map[string]interface{})["privkey"]
	if privkey != info.EncodedPrivKey {
		t.Errorf("setup didn't create correct private key")
	}
}

// Test that setup will fail if passed an invalid username from stdin
func TestSetupInvalidStdinResponse(t *testing.T) {
	ctx := context.Background()

	qriHome := createTmpQriHome(t)
	stdinText := "_not_valid_name"
	cmd, shutdown := newCommandWithStdin(ctx, qriHome, stdinText, repotest.NewTestCrypto())

	cmdText := "qri setup"
	err := executeCommand(cmd, cmdText)
	if err != nil {
		timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
	}
	if err == nil {
		t.Fatal("expected setup to fail due to invalid username, did not get an error")
	}
	expectErr := `username must start with a lower-case letter, and only contain lower-case letters, numbers, dashes, and underscores`
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect %s, got %s", expectErr, err.Error())
	}
}

// Test that setup will fail if passed an invalid username from the --username flag
func TestSetupInvalidUsernameFlag(t *testing.T) {
	ctx := context.Background()

	qriHome := createTmpQriHome(t)
	cmd, shutdown := newCommand(ctx, qriHome, repotest.NewTestCrypto())

	cmdText := "qri setup --username InvalidUser"
	err := executeCommand(cmd, cmdText)
	if err != nil {
		timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
	}
	if err == nil {
		t.Fatal("expected setup to fail due to invalid username, did not get an error")
	}
	expectErr := `username may not contain any upper-case letters`
	if expectErr != err.Error() {
		t.Errorf("error mismatch, expect %s, got %s", expectErr, err.Error())
	}
}

// Test that setup with the QRI_SETUP_CONFIG_DATA envvar will use that username
func TestSetupConfigData(t *testing.T) {
	ctx := context.Background()
	info := test_peers.GetTestPeerInfo(0)

	qriHome := createTmpQriHome(t)
	cmd, shutdown := newCommand(ctx, qriHome, repotest.NewTestCrypto())

	os.Setenv("QRI_SETUP_CONFIG_DATA", `{"profile":{"peername":"qri_my_fav_user"}}`)
	defer os.Setenv("QRI_SETUP_CONFIG_DATA", "")

	cmdText := "qri setup"
	if err := executeCommand(cmd, cmdText); err != nil {
		timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
		t.Fatal(err)
	}

	configData := readConfigFile(t, qriHome)

	username := configData["Profile"].(map[string]interface{})["peername"]
	expect := "qri_my_fav_user"
	if username != expect {
		t.Errorf("setup didn't create correct username, expect: %s, got: %s", expect, username)
	}

	profileID := configData["Profile"].(map[string]interface{})["id"]
	if profileID != info.EncodedPeerID {
		t.Errorf("setup didn't create correct profileID, expect: %s, got: %s", info.EncodedPeerID, profileID)
	}

	privkey := configData["Profile"].(map[string]interface{})["privkey"]
	if privkey != info.EncodedPrivKey {
		t.Errorf("setup didn't create correct private key")
	}
}

// Test that setup also works with real crypto
func TestSetupRealCrypto(t *testing.T) {
	ctx := context.Background()

	qriHome := createTmpQriHome(t)
	// NOTE: This test uses a real instance of the crypto generator. This will
	// be noticably slower than the other tests, and will also act non-deterministically.
	// Do not use this function in other tests, it is only used here to explicitly
	// verify that it works in this one case.
	cmd, shutdown := newCommand(ctx, qriHome, key.NewCryptoSource())

	cmdText := "qri setup"
	if err := executeCommand(cmd, cmdText); err != nil {
		timedShutdown(fmt.Sprintf("ExecCommand: %q\n", cmdText), shutdown)
		t.Fatal(err)
	}

	configData := readConfigFile(t, qriHome)

	username := configData["Profile"].(map[string]interface{})["peername"].(string)
	// NOTE: Shortest name I've ever seen is `snow_puli`
	if len(username) < 9 {
		t.Errorf("setup used real crypto, username seems wrong, got %s", username)
	}

	profileID := configData["Profile"].(map[string]interface{})["id"].(string)
	if len(profileID) != 46 {
		t.Errorf("setup used real crypto, profileID seems wrong, len is %d", len(profileID))
	}

	privkey := configData["Profile"].(map[string]interface{})["privkey"].(string)
	if len(privkey) != 1592 && len(privkey) != 1596 && len(privkey) != 1600 {
		t.Errorf("setup used real crypto, privkey seems wrong, len is %d", len(privkey))
	}
}

func createTmpQriHome(t *testing.T) string {
	tmpPath, err := ioutil.TempDir("", "qri_test_setup")
	if err != nil {
		t.Fatal(err)
	}

	qriHome := filepath.Join(tmpPath, "qri_home")
	err = os.MkdirAll(qriHome, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Setenv("QRI_PATH", qriHome)
	if err != nil {
		t.Fatal(err)
	}

	// Clear this envvar in order to get tests to pass on continuous build. If this
	// envvar is set, it will override other configuration.
	err = os.Setenv("QRI_SETUP_CONFIG_DATA", "")
	if err != nil {
		t.Fatal(err)
	}

	return qriHome
}

func readConfigFile(t *testing.T, path string) map[string]interface{} {
	configContents, err := ioutil.ReadFile(filepath.Join(path, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	configData := make(map[string]interface{})
	err = yaml.Unmarshal(configContents, &configData)
	if err != nil {
		t.Fatal(err)
	}

	return configData
}

func newCommand(ctx context.Context, path string, generator key.CryptoGenerator) (*cobra.Command, func() <-chan error) {
	return newCommandWithStdin(ctx, path, "", generator)
}

func newCommandWithStdin(ctx context.Context, path, stdinText string, generator key.CryptoGenerator) (*cobra.Command, func() <-chan error) {
	streams, in, _, _ := ioes.NewTestIOStreams()
	if stdinText != "" {
		in.WriteString(stdinText)
	}
	cmd, shutdown := NewQriCommand(ctx, path, generator, streams)
	cmd.SetOutput(streams.Out)
	return cmd, shutdown
}
