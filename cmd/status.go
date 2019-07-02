package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
	"github.com/qri-io/fill"
)

// NewStatusCommand creates new `qri status` command that statuss datasets for the local peer & others
func NewStatusCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &StatusOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"ls"},
		Short:   "Show current dataset status",
		Long: ``,
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

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json]")

	return cmd
}

// StatusOptions encapsulates state for the Status command
type StatusOptions struct {
	ioes.IOStreams

	Format          string
	Selection string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *StatusOptions) Complete(f Factory, args []string) (err error) {
	o.Selection = PwdSelection()
	if o.Selection == "" {
		return fmt.Errorf("this is not a qri directory")
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
		return err
	}

	dirData, err := ioutil.ReadFile("dataset.yaml")
	if err != nil {
		return err
	}

	if !bytes.Equal(res.Bytes, dirData) {
		printErr(o.ErrOut, fmt.Errorf("dataset.yaml has been modified"))
		return nil
	}

	printSuccess(o.ErrOut, "working directory clean!")
	return nil
}
