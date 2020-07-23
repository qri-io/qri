package cmd

import (
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	reporef "github.com/qri-io/qri/repo/ref"
	"github.com/spf13/cobra"
)

// NewPullCommand creates an add command
func NewPullCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &PullOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "pull DATASET [DATASET...]",
		Aliases: []string{"add"},
		Short:   "fetch & store datasets from other peers",
		Annotations: map[string]string{
			"group": "dataset",
		},
		Long: `Pull retrieves datasets owned by other peers and adds them to your repo. 
The reference names of the datasets will remain the same, including 
the name of the peer that originally added the dataset. You must have 
` + "`qri connect`" + ` running in another terminal to use this command.`,
		Example: `  # Pull a dataset named their_data, owned by other_peer:
  $ qri pull other_peer/their_data`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f); err != nil {
				return err
			}
			return o.Run(args)
		},
	}

	cmd.Flags().StringVar(&o.LinkDir, "link", "", "path to directory to link dataset to")
	cmd.MarkFlagFilename("link")
	cmd.Flags().BoolVar(&o.LogsOnly, "logs-only", false, "only fetch logs, skipping HEAD data")

	return cmd
}

// PullOptions encapsulates state for the add command
type PullOptions struct {
	ioes.IOStreams
	LinkDir        string
	LogsOnly       bool
	DatasetMethods *lib.DatasetMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *PullOptions) Complete(f Factory) (err error) {
	if o.DatasetMethods, err = f.DatasetMethods(); err != nil {
		return
	}
	return nil
}

// Run adds another peer's dataset to this user's repo
func (o *PullOptions) Run(args []string) error {

	if len(args) == 0 {
		return fmt.Errorf("nothing to pull")
	}
	if len(args) > 1 && o.LinkDir != "" {
		return fmt.Errorf("link flag can only be used with a single reference")
	}

	for _, arg := range args {
		p := &lib.PullParams{
			Ref:      arg,
			LinkDir:  o.LinkDir,
			LogsOnly: o.LogsOnly,
		}

		res := &dataset.Dataset{}
		if err := o.DatasetMethods.Pull(p, res); err != nil {
			return err
		}

		asRef := reporef.DatasetRef{
			Peername: res.Peername,
			Name:     res.Name,
			Path:     res.Path,
			Dataset:  res,
		}

		refStr := refStringer(asRef)
		fmt.Fprintf(o.Out, "\n%s", refStr.String())
	}

	return nil
}
