package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	regmock "github.com/qri-io/registry/regserver/mock"
)

func TestTutorialSkylarkTransformSecOne(t *testing.T) {
	testCommandSection(t, []subSection{
		subSection{
			name: "1.0",
			files: map[string]string{
				"dataset.yaml": `name: hello_world
meta:
  title: hello world example
transform:
  scriptpath: $PATH/transform.sky
`,
				"transform.sky": `
def transform(qri):
  return(["hello", "world"])
`,
			},
			commands: map[string]string{
				"qri add --file=$PATH/dataset.yaml": "",
				"qri info me/hello_world":           "",
				"qri data me/hello_world":           "",
				"qri data -s 1 me/hello_world":      "",
			},
		},

		subSection{
			name: "1.1",
			files: map[string]string{
				"transform.sky": `
def transform(qri):
  qri.set_meta("description", "this is an example dataset to learn about transformations")
  return(["hello","world"])
      `,
			},
			commands: map[string]string{
				"qri update --file=$PATH/dataset.yaml": "",
				"qri info me/hello_world":              "",
				"qri log me/hello_world":               "",
			},
		},
	})
}

type subSection struct {
	name            string
	commands, files map[string]string
}

func testCommandSection(t *testing.T, subSections []subSection) {
	t.Skip("not yet finished")

	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	_, registryServer := regmock.NewMockServer()

	path := filepath.Join(os.TempDir(), "qri_test_section")
	t.Logf("test filepath: %s", path)
	// fmt.Println(path)

	//clean up if previous cleanup failed
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.RemoveAll(path)
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Errorf("error creating test path: %s", err.Error())
		return
	}
	defer os.RemoveAll(path)

	// fmt.Printf("temp path: %s", path)
	os.Setenv("IPFS_PATH", filepath.Join(path, "ipfs"))
	os.Setenv("QRI_PATH", filepath.Join(path, "qri"))

	_, in, out, errs := NewTestIOStreams()
	root := NewQriCommand(NewDirPathFactory(path), in, out, errs)

	// run setup
	setup := fmt.Sprintf("setup --peername=alan --registry=%s", registryServer.URL)
	if _, err := executeCommand(root, strings.Split(setup, " ")...); err != nil {
		t.Fatal(err.Error())
	}

	// initializeCLI()
	// loadConfig()

	for _, ss := range subSections {
		for name, data := range ss.files {
			data = strings.Replace(data, "$PATH", path, -1)
			ioutil.WriteFile(filepath.Join(path, name), []byte(data), os.ModePerm)
		}

		for cmd := range ss.commands {
			cmd = strings.Replace(cmd, "$PATH", path, -1)
			cmd = strings.TrimPrefix(cmd, "qri ")
			_, err := executeCommand(root, strings.Split(cmd, " ")...)
			if err != nil {
				t.Fatal(err.Error())
			}
		}

	}

}
