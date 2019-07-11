package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewStatusCommand creates a `qri status` command that compares working directory to prev version
func NewStatusCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &StatusOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Show status of working directory",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	return cmd
}

// StatusOptions encapsulates state for the Status command
type StatusOptions struct {
	ioes.IOStreams

	Selection string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *StatusOptions) Complete(f Factory, args []string) (err error) {
	var ok bool
	o.Selection, ok = GetLinkedFilesysRef()
	if !ok {
		return fmt.Errorf("this is not a linked working directory")
	}
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the status command
func (o *StatusOptions) Run() (err error) {
	p := lib.GetParams{
		Path:     o.Selection,
		Selector: "",
	}
	res := lib.GetResult{}
	if err = o.DatasetRequests.Get(&p, &res); err != nil {
		printErr(o.ErrOut, fmt.Errorf("no previous version of this dataset"))
		printErr(o.ErrOut, fmt.Errorf("meta.json has modifications"))
		printErr(o.ErrOut, fmt.Errorf("schema.json has modifications"))
		// TODO(dlong): Output status of body
		printSuccess(o.ErrOut, "run `qri save` to commit a new version")
		return nil
	}

	isWorkingDirClean := true

	// Check status of meta component
	clean, err := checkCleanStatus(o.ErrOut, "meta.json", res.Dataset.Meta)
	isWorkingDirClean = isWorkingDirClean && clean
	if err != nil {
		return err
	}

	// Check status of schema component
	clean, err = checkCleanStatus(o.ErrOut, "schema.json", res.Dataset.Structure.Schema)
	isWorkingDirClean = isWorkingDirClean && clean
	if err != nil {
		return err
	}

	// TODO(dlong): Check status of body

	// Done, are we clean?
	if isWorkingDirClean {
		printSuccess(o.ErrOut, "working directory clean!")
	} else {
		printSuccess(o.ErrOut, "run `qri save` to commit a new version")
	}
	return nil
}

func checkCleanStatus(w io.Writer, localFilename string, component interface{}) (bool, error) {
	localData, err := ioutil.ReadFile(localFilename)
	if err != nil {
		return false, err
	}
	// Cleanup a bit.
	// TODO(dlong): Ignore whitespace changes, by parsing to a map[string] then reserializing.
	if localData[len(localData)-1] == '\n' {
		localData = localData[:len(localData)-1]
	}

	compData, err := json.MarshalIndent(component, "", " ")
	if err != nil {
		return false, err
	}
	if compData[len(compData)-1] == '\n' {
		compData = compData[:len(compData)-1]
	}

	if !bytes.Equal(localData, compData) {
		printErr(w, fmt.Errorf("%s has modifications", localFilename))
		return false, nil
	}
	return true, nil
}
