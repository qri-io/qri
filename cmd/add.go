package cmd

import (
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	reporef "github.com/qri-io/qri/repo/ref"
	"github.com/spf13/cobra"
)

// NewAddCommand creates an add command
func NewAddCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &AddOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "add DATASET [DATASET...]",
		Short: "add datasets from other peers",
		Annotations: map[string]string{
			"group": "dataset",
		},
		Long: `Add retrieves datasets owned by other peers and adds them to your repo. 
The reference names of the datasets will remain the same, including 
the name of the peer that originally added the dataset. You must have 
` + "`qri connect`" + ` running in another terminal to use this command.`,
		Example: `  # Add a dataset named their_data, owned by other_peer:
  $ qri add other_peer/their_data`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f); err != nil {
				return err
			}
			return o.Run(args)
		},
	}

	cmd.Flags().StringVar(&o.LinkDir, "link", "", "path to directory to link dataset to")
	cmd.Flags().BoolVar(&o.LogsOnly, "logs-only", false, "only fetch logs, skipping HEAD data")

	return cmd
}

// AddOptions encapsulates state for the add command
type AddOptions struct {
	ioes.IOStreams
	LinkDir         string
	LogsOnly        bool
	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *AddOptions) Complete(f Factory) (err error) {
	if o.DatasetRequests, err = f.DatasetRequests(); err != nil {
		return
	}
	return nil
}

// Run adds another peer's dataset to this user's repo
func (o *AddOptions) Run(args []string) error {
	o.StartSpinner()
	defer o.StopSpinner()

	if len(args) == 0 {
		return fmt.Errorf("nothing to add")
	}
	if len(args) > 1 && o.LinkDir != "" {
		return fmt.Errorf("link flag can only be used with a single reference")
	}

	for _, arg := range args {
		p := &lib.AddParams{
			Ref:      arg,
			LinkDir:  o.LinkDir,
			LogsOnly: o.LogsOnly,
		}

		res := reporef.DatasetRef{}
		if err := o.DatasetRequests.Add(p, &res); err != nil {
			return err
		}

		refStr := refStringer(res)
		fmt.Fprintf(o.Out, "\n%s", refStr.String())
		printInfo(o.Out, "Successfully added dataset %s", arg)
	}

	return nil
}
