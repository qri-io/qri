package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewInfoCommand creates a `qri info` cobra command for describing datasets
func NewInfoCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &InfoOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "info",
		Aliases: []string{"get", "describe"},
		Short:   "Show summarized description of a dataset",
		Long: `Info describes datasets. By default, it will return the peername, dataset name, 
the network, the dataset hash, the file size, the length of the datasets, 
and the validation errors.

Using the ` + "`--format`" + ` flag, you can get output in json. This will return a json
representation of the dataset, without the dataset body, identical to 
` + "`qri get --format json`" + `.

To get info on a peer's dataset, you must be running ` + "`qri connect`" + ` in a separate 
terminal window.`,
		Example: `  # get info for my dataset:
  qri info me/annual_pop

  # get info for a dataset at a specific version:
  qri info me@/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn

  or

  qri info me/comics@/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn

  # get info in json format
  qri info -f json me/annual_pop

  # to get info on a peer's dataset, spin up your qri node
  qri connect

  # then, in a separate window, request the info from peer b5
  qri info b5/comics`,
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

// InfoOptions encapsulates state for the info command
type InfoOptions struct {
	ioes.IOStreams

	Refs   []string
	Format string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *InfoOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the info command
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

	if len(o.Refs) == 0 {
		return o.info(0, "")
	}

	for i, refstr := range o.Refs {
		if err := o.info(i, refstr); err != nil {
			return err
		}
	}

	return nil
}

// info prints terse information about a single dataset
func (o *InfoOptions) info(index int, refstr string) error {
	ref, err := repo.ParseDatasetRef(refstr)
	if err != nil && err != repo.ErrEmptyRef {
		return err
	}

	if ref.IsPeerRef() {
		return fmt.Errorf("please specify a dataset for peer %s", ref.Peername)
	}

	p := lib.LookupParams{
		Ref: &ref,
	}
	res := lib.LookupResult{}
	if err = o.DatasetRequests.Get(&p, &res); err != nil {
		if err == repo.ErrEmptyRef {
			return lib.NewError(err, "please provide a dataset reference")
		}
		return err
	}

	if o.Format == "" {
		printDatasetRefInfo(o.Out, index, *p.Ref)
	} else {
		data, err := json.MarshalIndent(res.Data, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(o.Out, "%s", string(data))
	}
	return nil
}
