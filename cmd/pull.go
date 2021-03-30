package cmd

import (
	"context"
	"fmt"

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
		Long: `Pull downloads datasets and stores them locally, fetching the dataset log and
dataset version(s). By default pull fetches the latest version of a dataset.
`,
		Example: `  # download a dataset log and latest version
  $ qri pull b5/world_bank_population

  # pull a specific version from a remote by hash
  $ qri pull ramfox b5/world_bank_population@/ipfs/QmFoo...`,
		Annotations: map[string]string{
			"group": "network",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f); err != nil {
				return err
			}
			return o.Run(args)
		},
	}

	cmd.Flags().StringVar(&o.LinkDir, "link", "", "path to directory to link dataset to")
	cmd.Flags().StringVar(&o.Remote, "remote", "", "location to pull from")
	cmd.MarkFlagFilename("link")
	cmd.Flags().BoolVar(&o.LogsOnly, "logs-only", false, "only fetch logs, skipping HEAD data")

	return cmd
}

// PullOptions encapsulates state for the add command
type PullOptions struct {
	ioes.IOStreams
	LinkDir  string
	Remote   string
	LogsOnly bool

	inst *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *PullOptions) Complete(f Factory) (err error) {
	o.inst, err = f.Instance()
	return
}

// Run adds another peer's dataset to this user's repo
func (o *PullOptions) Run(args []string) error {

	if len(args) == 0 {
		return fmt.Errorf("nothing to pull")
	}
	if len(args) > 1 && o.LinkDir != "" {
		return fmt.Errorf("link flag can only be used with a single reference")
	}

	ctx := context.TODO()

	for _, arg := range args {
		p := &lib.PullParams{
			Ref:      arg,
			LinkDir:  o.LinkDir,
			LogsOnly: o.LogsOnly,
			Remote:   o.Remote,
		}

		res, err := o.inst.WithSource("network").Dataset().Pull(ctx, p)
		if err != nil {
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
