package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewStatsCommand creates a new `qri search` command that searches for datasets
func NewStatsCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &StatsOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Get aggregated stats for a dataset",
		Long: `
Run the ` + "`stats`" + ` to generate and view stats for a dataset using a dataset reference.`,
		Example: `  # get stats for me/dataset_name:
  qri stats me/dataset_name`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				fmt.Println("errorrrr")
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().BoolVarP(&o.Pretty, "pretty", "p", true, "clear the current selection")

	return cmd
}

// StatsOptions encapsulates state for the search command
type StatsOptions struct {
	ioes.IOStreams

	Ref    string
	Pretty bool

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *StatsOptions) Complete(f Factory, args []string) (err error) {
	if o.DatasetRequests, err = f.DatasetRequests(); err != nil {
		return
	}

	if len(args) < 1 {
		return fmt.Errorf("need a dataset reference, eg: me/dataset_name")
	}
	o.Ref = args[0]
	return
}

// Validate checks that any user input is valide
func (o *StatsOptions) Validate() error {
	return nil
}

// Run executes the search command
func (o *StatsOptions) Run() (err error) {
	p := &lib.StatsParams{Ref: o.Ref}
	r := &lib.StatsResponse{}
	if err = o.DatasetRequests.Stats(p, r); err != nil {
		return err
	}

	statsBytes, err := ioutil.ReadAll(r.Reader)
	if err != nil {
		return fmt.Errorf("error reading stats")
	}
	// if o.Pretty {
	// some nicely formatted stats
	// }
	statsBytes = append(statsBytes, byte('\n'))
	printInfo(o.Out, string(statsBytes))
	return nil
}
