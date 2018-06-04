package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qri-io/qri/config"
	regmock "github.com/qri-io/registry/regserver/mock"
	"github.com/spf13/cobra"
)

func confirmQriNotRunning() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", config.DefaultAPIPort))
	if err != nil {
		return fmt.Errorf("it looks like a qri server is already running on port %d, please close before running tests", config.DefaultAPIPort)
	}

	l.Close()
	return nil
}

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

const linksJSONData = `[
  "http://datatogether.org",
  "https://datatogether.org/css/style.css",
  "https://datatogether.org/img/favicon.ico",
  "https://datatogether.org",
  "https://datatogether.org/public-record",
  "https://datatogether.org/activities",
  "https://datatogether.org/activities/harvesting",
  "https://datatogether.org/activities/monitoring",
  "https://datatogether.org/activities/storing",
  "https://datatogether.org/activities/rescuing",
  "http://2017.code4lib.org",
  "https://datatogether.org/presentations/Code4Lib%202017%20-%20Golden%20Age%20for%20Libraries%20-%20Storing%20Data%20Together.pdf",
  "https://datatogether.org/presentations/Code4Lib%202017%20-%20Golden%20Age%20for%20Libraries%20-%20Storing%20Data%20Together.key",
  "http://www.esipfed.org/meetings/upcoming-meetings/esip-summer-meeting-2017",
  "https://datatogether.org/presentations/Data%20Together%20-%20ESIP%20Summer%20Meeting%20July%202017.pdf",
  "https://datatogether.org/presentations/Data%20Together%20-%20ESIP%20Summer%20Meeting%20July%202017.key",
  "https://archive.org/details/ndsr-dc-2017",
  "https://datatogether.org/presentations/Data%20Together%20-%20NDSR%20-%20swadeshi.pdf",
  "https://datatogether.org/presentations/Data%20Together%20-%20NDSR%20-%20swadeshi.key",
  "https://github.com/datatogether"
]`

const profileData = `
{
	"description" : "I'm a description!"
}
`

// This is a basic integration test that makes sure basic happy paths work on the CLI
func TestCommandsIntegration(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	_, registryServer := regmock.NewMockServer()

	path := filepath.Join(os.TempDir(), "qri_test_commands_integration")
	t.Logf("test filepath: %s", path)
	fmt.Println(path)

	// fmt.Printf("temp path: %s", path)
	os.Setenv("IPFS_PATH", filepath.Join(path, "ipfs"))
	os.Setenv("QRI_PATH", filepath.Join(path, "qri"))

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

	linksFilepath := filepath.Join(path, "/links.json")
	if err := ioutil.WriteFile(linksFilepath, []byte(linksJSONData), os.ModePerm); err != nil {
		t.Errorf("error writing json file: %s", err.Error())
		return
	}

	profileDataFilepath := filepath.Join(path, "/profile")
	if err := ioutil.WriteFile(profileDataFilepath, []byte(profileData), os.ModePerm); err != nil {
		t.Errorf("error profile json file: %s", err.Error())
		return
	}

	commands := [][]string{
		{"help"},
		{"version"},
		{"setup", "--peername=" + "alan", "--registry=" + registryServer.URL},
		{"config", "get", "-c"},
		{"config", "get", "profile"},
		{"config", "set", "webapp.port", "3505"},
		// TODO - add setting whole config via a file
		// {"config", "set", "-i" + profileDataFilepath},
		{"info", "me"},
		{"add", "--data=" + moviesFilePath, "me/movies"},
		{"add", "--data=" + movies2FilePath, "me/movies2"},
		{"add", "--data=" + linksFilepath, "me/links"},
		{"info", "me/movies"},
		{"list"},
		{"save", "--data=" + movies2FilePath, "-t" + "commit_1", "me/movies"},
		{"log", "me/movies"},
		{"diff", "me/movies", "me/movies2", "-d", "detail"},
		{"export", "--dataset", "-o" + path, "me/movies"},
		{"export", "--all", "-o" + path, "--format=json", "--data-format=json", "me/movies"},
		{"rename", "me/movies", "me/movie"},
		{"data", "--limit=1", "--data-format=cbor", "me/movie"},
		{"validate", "me/movie"},
		{"remove", "me/movie"},
		{"export", "--blank", "-o" + path + "/blank_dataset.yaml"},
		{"setup", "--remove"},
	}

	for i, args := range commands {
		func() {
			defer func() {
				if e := recover(); e != nil {
					t.Errorf("case %d unexpected panic executing command\n%s\n%s", i, strings.Join(args, " "), e)
					return
				}
			}()
			_, err := executeCommand(RootCmd, args...)
			if err != nil {
				t.Errorf("case %d unexpected error executing command\n%s\n%s", i, strings.Join(args, " "), err.Error())
				return
			}
		}()
	}
}
