package cmd

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewDataCommand creates a new `qri data` cobra command to fetch entries from the body of a dataset
func NewDataCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &DataOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "data",
		Short: "Read dataset data",
		Long: `
Data reads records from a dataset`,
		Example: `  show the first 50 rows of a dataset:
  $ qri data me/dataset_name`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Run())
		},
	}

	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "path to write to, default is stdout")
	cmd.Flags().BoolVarP(&o.All, "all", "a", false, "read all dataset entries (overrides limit, offest)")
	cmd.Flags().StringVarP(&o.Format, "data-format", "f", "json", "format to export. one of [json,csv,cbor]")
	cmd.Flags().IntVarP(&o.Limit, "limit", "l", 50, "max number of records to read")
	cmd.Flags().IntVarP(&o.Offset, "offset", "s", 0, "number of records to skip")

	return cmd
}

// DataOptions encapsulates options for the data command
type DataOptions struct {
	IOStreams

	Format string
	Output string
	Limit  int
	Offset int
	All    bool
	Ref    string

	UsingRPC        bool
	DatasetRequests *core.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *DataOptions) Complete(f Factory, args []string) (err error) {
	o.Ref = args[0]
	o.UsingRPC = f.RPC() != nil
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the data command
func (o *DataOptions) Run() error {
	if o.UsingRPC {
		return usingRPCError("data")
	}

	dsr, err := repo.ParseDatasetRef(o.Ref)
	if err != nil {
		return err
	}

	res := &repo.DatasetRef{}
	if err = o.DatasetRequests.Get(&dsr, res); err != nil {
		return err
	}
	ds := res.Dataset
	df, err := dataset.ParseDataFormatString(o.Format)
	if err != nil {
		return err
	}

	p := &core.LookupParams{
		Format: df,
		Path:   ds.Path,
		Limit:  o.Limit,
		Offset: o.Offset,
		All:    o.All,
	}

	result := &core.LookupResult{}
	if err := o.DatasetRequests.LookupBody(p, result); err != nil {
		return err
	}

	data := result.Data
	if p.Format == dataset.CBORDataFormat {
		data = []byte(hex.EncodeToString(result.Data))
	}

	if o.Output != "" {
		ioutil.WriteFile(o.Output, data, os.ModePerm)
	} else {
		fmt.Fprintln(o.Out, string(data))
	}

	return nil
}
