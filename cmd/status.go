package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/qri/fsi"
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
	Dir string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *StatusOptions) Complete(f Factory, args []string) (err error) {
	var ok bool
	
	o.Dir, err = os.Getwd()
	if err != nil {
		return err
	}

	o.Selection, ok = fsi.GetLinkedFilesysRef(o.Dir)
	if !ok {
		return fmt.Errorf("this is not a linked working directory")
	}
	
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the status command
func (o *StatusOptions) Run() (err error) {
	res := []lib.StatusItem{}
	if err := o.DatasetRequests.Status(&o.Dir,&res); err != nil {
		printErr(o.ErrOut, err)
		return nil
	}

	if len(res) == 0 {
		printSuccess(o.ErrOut, "working directory clean!")
		return nil
	}

	for _, si := range res {
		printErr(o.ErrOut, fmt.Errorf("%s: %s (source: %s)", si.Type, si.Path, filepath.Base(si.SourceFile)))
	}

	return nil

	// p := lib.GetParams{
	// 	Path:     o.Selection,
	// 	Selector: "",
	// }
	// res := lib.GetResult{}
	// if err = o.DatasetRequests.Get(&p, &res); err != nil {
	// 	printErr(o.ErrOut, fmt.Errorf("no previous version of this dataset"))
	// 	printErr(o.ErrOut, fmt.Errorf("meta.json has modifications"))
	// 	printErr(o.ErrOut, fmt.Errorf("schema.json has modifications"))
	// 	// TODO(dlong): Output status of body
	// 	printSuccess(o.ErrOut, "run `qri save` to commit a new version")
	// 	return nil
	// }

	// wd, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }

	// if _, _, err = fsi.ReadDir(wd); err != nil {
	// 	return err
	// }

	// isWorkingDirClean := true

	// Check status of meta component
	// clean, err := checkCleanStatus(o.ErrOut, "meta.json", res.Dataset.Meta)
	// isWorkingDirClean = isWorkingDirClean && clean
	// if err != nil {
	// 	return err
	// }

	// // Check status of schema component
	// clean, err = checkCleanStatus(o.ErrOut, "schema.json", res.Dataset.Structure.Schema)
	// isWorkingDirClean = isWorkingDirClean && clean
	// if err != nil {
	// 	return err
	// }

	// TODO(dlong): Check status of body

	// Done, are we clean?
	// if isWorkingDirClean {
	// 	printSuccess(o.ErrOut, "working directory clean!")
	// } else {
	// 	printSuccess(o.ErrOut, "run `qri save` to commit a new version")
	// }
	return nil
}