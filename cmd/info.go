package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewInfoCommand creates a `qri info` cobra command for describing datasets
func NewInfoCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &InfoOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "info",
		Aliases: []string{"get", "describe"},
		Short:   "show summarized description of a dataset",
		Long:    `info describes datasets`,
		Example: `  get info for b5/comics:
  $ qri info b5/comics

  get info for a dataset at a specific version:
  $ qri info me@/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn

  or

  $ qri info me/comics@/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Run())
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json]")
	return cmd
}

// InfoOptions encapsulates state for the info command
type InfoOptions struct {
	IOStreams

	Refs   []string
	Format string

	DatasetRequests *core.DatasetRequests
}

func (o *InfoOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

func (o *InfoOptions) Run() error {
	if o.Format != "" {
		format, err := dataset.ParseDataFormatString(o.Format)
		if err != nil {
			return fmt.Errorf("invalid data format: %s", o.Format)
		}
		if format != dataset.JSONDataFormat {
			return fmt.Errorf("invalid data format. currently only json or plaintext are supported")
		}
	}

	for i, refstr := range o.Refs {
		ref, err := repo.ParseDatasetRef(refstr)
		if err != nil {
			return err
		}

		if ref.IsPeerRef() {
			printWarning(o.Out, "please specify a dataset for peer %s", ref.Peername)
		} else {
			res := repo.DatasetRef{}
			err = o.DatasetRequests.Get(&ref, &res)
			ExitIfErr(err)

			if o.Format == "" {
				printDatasetRefInfo(o.Out, i, res)
			} else {
				data, err := json.MarshalIndent(res.Dataset, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintf(o.Out, "%s", string(data))
			}
		}
	}

	return nil
}
