package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	_, output, err = executeCommandC(root, args...)
	return output, err
}

func executeCommandC(root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := &bytes.Buffer{}
	root.SetOutput(buf)
	root.SetArgs(args)

	c, err = root.ExecuteC()
	return c, buf.String(), err
}

const moviesCSVData = `movie_title,duration
Avatar,178
Pirates of the Caribbean: At World's End,169
Spectre,148
The Dark Knight Rises ,164
Star Wars: Episode VII - The Force Awakens,15
John Carter,132
Spider-Man 3,156
Tangled,100
Avengers: Age of Ultron,141`

// This is a basic integration test that makes sure basic happy paths work on the CLI
func TestCommandsIntegration(t *testing.T) {
	path := filepath.Join(os.TempDir(), "qri_test_commands_integration")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating test path: %s", err.Error())
		return
	}

	// defer os.RemoveAll(path)

	moviesFilePath := filepath.Join(path, "/movies.csv")
	if err := ioutil.WriteFile(moviesFilePath, []byte(moviesCSVData), os.ModePerm); err != nil {
		t.Errorf("error writing csv file: %s", err.Error())
		return
	}

	os.Setenv("IPFS_PATH", filepath.Join(path, "ipfs"))
	os.Setenv("QRI_PATH", filepath.Join(path, "qri"))

	commands := [][]string{
		{"help"},
		{"version"},
		{"init"},
		{"add", "-f" + moviesFilePath, "-n" + "movies"},
		// {"validate"},
		// {"update"},
		{"run", "select * from movies limit 5"},
		{"rename", "movies", "movie"},
		{"remove", "movie"},
	}

	for i, args := range commands {
		_, err := executeCommand(RootCmd, args...)
		if err != nil {
			t.Errorf("case %d unexpected error executing command: %s", i, err.Error())
			return
		}
	}
}
