package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewStatsCommand creates a new `qri stats` command that display stats of datasets
func NewStatsCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &StatsOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "stats DATASET",
		Short: "get aggregated stats for a dataset",
		Long:  `Run the ` + "`stats`" + ` to generate and view stats for a dataset using a dataset reference.`,
		Example: `  # Get stats for me/dataset_name:
  $ qri stats me/dataset_name`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().BoolVarP(&o.Pretty, "pretty", "p", false, "whether to print output with indentation")

	return cmd
}

// StatsOptions encapsulates state for the stats command
type StatsOptions struct {
	ioes.IOStreams

	Refs   *RefSelect
	Pretty bool

	DatasetMethods *lib.DatasetMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *StatsOptions) Complete(f Factory, args []string) (err error) {
	if o.DatasetMethods, err = f.DatasetMethods(); err != nil {
		return
	}

	o.Refs, err = GetCurrentRefSelect(f, args, 1, nil)
	if err != nil {
		return err
	}
	return
}

// Validate checks that any user input is valid
func (o *StatsOptions) Validate() error {
	return nil
}

// Run executes the stats command
func (o *StatsOptions) Run() (err error) {
	printRefSelect(o.ErrOut, o.Refs)

	p := &lib.StatsParams{Ref: o.Refs.Ref()}
	r := &lib.StatsResponse{}
	if err = o.DatasetMethods.Stats(p, r); err != nil {
		return err
	}

	var buffer []byte
	if !o.Pretty {
		buffer = append(r.StatsBytes, byte('\n'))
	} else {
		// stats is already in JSON format, so just needs pretty printing
		buf := new(bytes.Buffer)
		err = json.Indent(buf, []byte(r.StatsBytes), "", "  ")
		if err != nil {
			return fmt.Errorf("err encoding stats: %s", err)
		}
		buffer = buf.Bytes()
	}

	printInfo(o.Out, string(buffer))
	return nil
}
