package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewListCommand creates new `qri list` command that lists datasets for the local peer & others
func NewListCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ListOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Show a list of datasets",
		Long: `
List shows lists of datasets, including names and current hashes. 

The default list is the latest version of all datasets you have on your local 
qri repository.

When used in conjunction with ` + "`qri connect`" + `, list can list a peer's dataset. You
must have ` + "`qri connect`" + ` running in a separate terminal window.`,
		Example: `  # show all of your datasets:
  qri list

  # to view the list of your peer's dataset,
  # in one terminal window:
  qri connect

  # in a separate terminal window, to show all of b5's datasets:
  qri list b5`,
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
	cmd.Flags().IntVarP(&o.Limit, "limit", "l", 25, "limit results, default 25")
	cmd.Flags().IntVarP(&o.Offset, "offset", "o", 0, "offset results, default 0")
	cmd.Flags().BoolVarP(&o.Published, "published", "p", false, "list only published datasets")
	cmd.Flags().BoolVarP(&o.ShowNumVersions, "num-versions", "n", false, "show number of versions")

	return cmd
}

// ListOptions encapsulates state for the List command
type ListOptions struct {
	ioes.IOStreams

	Format          string
	Limit           int
	Offset          int
	Peername        string
	Published       bool
	ShowNumVersions bool

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

	refs := []repo.DatasetRef{}

	if o.Peername == "" {

		p := &lib.ListParams{
			Limit:           o.Limit,
			Offset:          o.Offset,
			Published:       o.Published,
			ShowNumVersions: o.ShowNumVersions,
		}
		if err = o.DatasetRequests.List(p, &refs); err != nil {
			return err
		}
	} else {
		// if user provides "me/my_dataset", split into peername="me" and name="my_dataset"
		peername := o.Peername
		dsName := ""
		parts := strings.Split(peername, "/")
		if len(parts) > 1 {
			peername = parts[0]
			dsName = parts[1]
		}
		// TODO: It would be a bit more efficient to pass dsName to the ListParams
		// and only retrieve information about that one dataset.
		p := &lib.ListParams{
			Peername:        peername,
			Limit:           o.Limit,
			Offset:          o.Offset,
			ShowNumVersions: o.ShowNumVersions,
		}
		if err = o.DatasetRequests.List(p, &refs); err != nil {
			return err
		}

		replace := make([]repo.DatasetRef, 0)
		for _, ref := range refs {
			// remove profileID so names print pretty
			ref.ProfileID = ""
			// if there's a dsName that restricts the list operation, append matches
			if dsName != "" && dsName == ref.Name {
				replace = append(replace, ref)
			}
		}

		// if there's a dsName that restricts the list operation, only show that dataset
		if dsName != "" {
			refs = replace
		}

		if len(refs) == 0 {
			if dsName != "" {
				printInfo(o.Out, "%s has no datasets that match \"%s\"", peername, dsName)
			} else {
				printInfo(o.Out, "%s has no datasets", peername)
			}
			return
		}
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

	return nil
}
