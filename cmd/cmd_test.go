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
	"time"

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

func executeCommand(root *cobra.Command, cmd string) (output string, err error) {
	cmd = strings.TrimPrefix(cmd, "qri ")
	// WARNING - currently doesn't support quoted strings as input
	args := strings.Split(cmd, " ")
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
	fmt.Println(path)
	t.Logf("test filepath: %s", path)

	//clean up if previous cleanup failed
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("failed to cleanup from previous test execution: %s", err.Error())
		}
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

	commands := []string{
		"qri help",
		"qri version",
		fmt.Sprintf("qri setup --peername=alan --registry=%s", registryServer.URL),
		"qri config get -c",
		"qri config get profile",
		"qri config set webapp.port 3505",
		"qri info me",
		fmt.Sprintf("qri add --body=%s me/movies", moviesFilePath),
		fmt.Sprintf("qri add --body=%s me/movies2", movies2FilePath),
		fmt.Sprintf("qri add --body=%s me/links", linksFilepath),
		"qri info me/movies",
		"qri list",
		fmt.Sprintf("qri save --body=%s -t=commit_1 me/movies", movies2FilePath),
		"qri log me/movies",
		"qri diff me/movies me/movies2 -d=detail",
		fmt.Sprintf("qri export -o=%s me/movies", path),
		fmt.Sprintf("qri export -o=%s --format=cbor --body-format=json me/movies", path),
		"qri registry unpublish me/movies",
		"qri registry publish me/movies",
		"qri rename me/movies me/movie",
		"qri body --limit=1 --format=cbor me/movie",
		"qri validate me/movie",
		"qri remove me/movie",
		fmt.Sprintf("qri export --blank -o=%s/blank_dataset.yaml", path),
		"qri setup --remove",
	}

	_, in, out, err := NewTestIOStreams()
	root := NewQriCommand(NewDirPathFactory(path), in, out, err)

	for i, command := range commands {
		func() {
			defer func() {
				if e := recover(); e != nil {
					t.Errorf("case %d unexpected panic executing command\n%s\n%s", i, command, e)
					return
				}
			}()
			_, err := executeCommand(root, command)
			time.Sleep(100 * time.Millisecond)
			if err != nil {
				t.Errorf("case %d unexpected error executing command\n%s\n%s", i, command, err.Error())
				return
			}
		}()
	}
}
