package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewListCommand creates new `qri list` command that lists datasets for the local peer & others
func NewListCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &ListOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "show a list of datasets",
		Long: `
list shows lists of datasets, including names and current hashes. 

The default list is the latest version of all datasets you have on your local 
qri repository.`,
		Example: `  show all of your datasets:
  $ qri list`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Run())
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json]")
	cmd.Flags().IntVarP(&o.Limit, "limit", "l", 25, "limit results, default 25")
	cmd.Flags().IntVarP(&o.Offset, "offset", "o", 0, "offset results, default 0")

	return cmd
}

// ListOptions encapsulates state for the List command
type ListOptions struct {
	IOStreams

	Format   string
	Limit    int
	Offset   int
	Peername string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ListOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		o.Peername = args[0]
	}
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the list command
func (o *ListOptions) Run() (err error) {
	if o.Peername == "" {

		p := &lib.ListParams{
			Limit:  o.Limit,
			Offset: o.Offset,
		}
		refs := []repo.DatasetRef{}
		if err = o.DatasetRequests.List(p, &refs); err != nil {
			return err
		}

		switch o.Format {
		case "":
			for i, ref := range refs {
				printDatasetRefInfo(o.Out, i+1, ref)
			}
		case dataset.JSONDataFormat.String():
			data, err := json.MarshalIndent(refs, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintf(o.Out, "%s\n", string(data))
		default:
			return fmt.Errorf("unrecognized format: %s", o.Format)
		}
	} else {

		p := &lib.ListParams{
			Peername: o.Peername,
			Limit:    o.Limit,
			Offset:   o.Offset,
		}
		refs := []repo.DatasetRef{}
		if err = o.DatasetRequests.List(p, &refs); err != nil {
			return err
		}

		for _, ref := range refs {
			// remove profileID so names print pretty
			ref.ProfileID = ""
		}

		switch o.Format {
		case "":
			if len(refs) == 0 {
				printInfo(o.Out, "%s has no datasets", o.Peername)
			} else {
				for i, ref := range refs {
					printDatasetRefInfo(o.Out, i+1, ref)
				}
			}
		case dataset.JSONDataFormat.String():
			data, err := json.MarshalIndent(refs, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintf(o.Out, "%s\n", string(data))
		default:
			return fmt.Errorf("unrecognized format: %s", o.Format)
		}
	}

	return nil
}
