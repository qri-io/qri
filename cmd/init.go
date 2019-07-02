package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewInitCommand creates new `qri init` command that connects a directory to
// the local filesystem
func NewInitCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &InitOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"ls"},
		Short:   "initialize a dataset directory",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Complete(f)
			return o.Run()
		},
	}

	cmd.PersistentFlags().StringVar(&o.Peername, "peername", "me", "ref peername")

	return cmd
}

// InitOptions encapsulates state for the List command
type InitOptions struct {
	ioes.IOStreams

	Peername        string
	DatasetRequests *lib.DatasetRequests
}

// Complete completes a dataset reference
func (o *InitOptions) Complete(f Factory) (err error) {
	o.DatasetRequests, err = f.DatasetRequests()
	return err
}

// Run executes the init command
func (o *InitOptions) Run() (err error) {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	ref := fmt.Sprintf("%s/%s", o.Peername, filepath.Base(pwd))
	path := "dataset.yaml"

	p := lib.GetParams{
		Path:     ref,
		Selector: "",
	}

	res := lib.GetResult{}
	if err = o.DatasetRequests.Get(&p, &res); err != nil {
		if err = ioutil.WriteFile(".qri_ref", []byte(ref), os.ModePerm); err != nil {
			return fmt.Errorf("creating dataset reference: %s", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := ioutil.WriteFile(path, []byte(blankYamlDataset), os.ModePerm); err != nil {
				return err
			}
			printSuccess(o.Out, "initialized qri dataset %s", path)
			return nil
		}
		return fmt.Errorf("'%s' already exists", path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := ioutil.WriteFile("dataset.yaml", res.Bytes, os.ModePerm); err != nil {
			return err
		}
	}

	ref = fmt.Sprintf("%s/%s@%s%s", o.Peername, filepath.Base(pwd), res.Dataset.Commit.Author.ID, res.Dataset.Path)
	return ioutil.WriteFile(".qri_ref", []byte(ref), os.ModePerm)
}
