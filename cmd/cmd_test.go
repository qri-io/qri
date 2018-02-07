package cmd

import (
	"bytes"
	"fmt"
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

const moviesCSVData2 = `movie_title,duration
Avatar,178
Pirates of the Caribbean: At World's End,169
Spectre,148
The Dark Knight Rises ,164
Star Wars: Episode VII - The Force Awakens,15
John Carter,132
Spider-Man 3,156
Tangled,100
Avengers: Age of Ultron,141
A Wild Film Appears!,2000
Another Film!,121`

const profileData = `
{
	"description" : "I'm a description!"
}
`

// This is a basic integration test that makes sure basic happy paths work on the CLI
func TestCommandsIntegration(t *testing.T) {
	path := filepath.Join(os.TempDir(), "qri_test_commands_integration")
	//clean up if previous cleanup failed
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.RemoveAll(path)
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating test path: %s", err.Error())
		return
	}
	defer os.RemoveAll(path)

	moviesFilePath := filepath.Join(path, "/movies.csv")
	if err := ioutil.WriteFile(moviesFilePath, []byte(moviesCSVData), os.ModePerm); err != nil {
		t.Errorf("error writing csv file: %s", err.Error())
		return
	}

	movies2FilePath := filepath.Join(path, "/movies2.csv")
	if err := ioutil.WriteFile(movies2FilePath, []byte(moviesCSVData2), os.ModePerm); err != nil {
		t.Errorf("error writing csv file: %s", err.Error())
		return
	}

	profileDataFilepath := filepath.Join(path, "/profile")
	if err := ioutil.WriteFile(profileDataFilepath, []byte(profileData), os.ModePerm); err != nil {
		t.Errorf("error profile json file: %s", err.Error())
		return
	}

	os.Setenv("IPFS_PATH", filepath.Join(path, "ipfs"))
	os.Setenv("QRI_PATH", filepath.Join(path, "qri"))

	t.Log("PATH:", path)
	fmt.Println(path)

	commands := [][]string{
		{"help"},
		{"version"},
		{"setup", "--peername=" + "alan"},
		{"profile", "get"},
		{"profile", "set", "-f" + profileDataFilepath},
		{"config", "get"},
		{"info"},
		{"add", "-v", "--data=" + moviesFilePath, "me/movies"},
		{"add", "--data=" + movies2FilePath, "me/movies2"},
		{"list"},
		{"save", "--data=" + movies2FilePath, "-t" + "commit_1", "me/movies"},
		{"log", "me/movies"},
		{"diff", "me/movies", "me/movies2", "-d", "detail"},
		{"export", "--dataset", "me/movies", "-o" + path},
		{"rename", "me/movies", "me/movie"},
		{"validate", "me/movie"},
		{"remove", "me/movie"},
	}

	for i, args := range commands {
		_, err := executeCommand(RootCmd, args...)
		if err != nil {
			t.Errorf("case %d unexpected error executing command: %s", i, err.Error())
			return
		}
	}
}
