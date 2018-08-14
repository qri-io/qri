package cmd

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewBodyCommand creates a new `qri body` cobra command to fetch entries from the body of a dataset
func NewBodyCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &BodyOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "body",
		Short: "Get the body of a dataset",
		Long: `
'qri body' reads records from a dataset`,
		Example: `  show the first 50 rows of a dataset:
  $ qri body me/dataset_name

  show the next 50 rows of a dataset:
  $ qri body --offset 50 me/dataset_name

  save the body as csv to file
  $ qri body -o new_file.csv -f csv me/dataset_name`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "path to write to, default is stdout")
	cmd.Flags().BoolVarP(&o.All, "all", "a", false, "read all dataset entries (overrides limit, offest)")
	cmd.Flags().StringVarP(&o.Format, "format", "f", "json", "format to export. one of [json,csv,cbor]")
	cmd.Flags().IntVarP(&o.Limit, "limit", "l", 50, "max number of records to read")
	cmd.Flags().IntVarP(&o.Offset, "offset", "s", 0, "number of records to skip")

	return cmd
}

// BodyOptions encapsulates options for the body command
type BodyOptions struct {
	IOStreams

	Format string
	Output string
	Limit  int
	Offset int
	All    bool
	Ref    string

	UsingRPC        bool
	DatasetRequests *lib.DatasetRequests
	Repo            repo.Repo
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *BodyOptions) Complete(f Factory, args []string) (err error) {
	o.Ref = args[0]
	o.UsingRPC = f.RPC() != nil
	o.DatasetRequests, err = f.DatasetRequests()
	if err != nil {
		return
	}
	o.Repo, err = f.Repo()
	return
}

// Run executes the body command
func (o *BodyOptions) Run() error {
	if o.UsingRPC {
		return usingRPCError("body")
	}

	dsr, err := repo.ParseDatasetRef(o.Ref)
	if err != nil {
		return err
	}

	if err = lib.DefaultSelectedRef(o.Repo, &dsr); err != nil {
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

	p := &lib.LookupParams{
		Format: df,
		Path:   ds.Path,
		Limit:  o.Limit,
		Offset: o.Offset,
		All:    o.All,
	}

	result := &lib.LookupResult{}
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
