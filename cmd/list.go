package cmd

import (
	"encoding/json"
	"fmt"

	util "github.com/datatogether/api/apiutil"
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
qri repository. The first argument can be used to find datasets with a certain 
substring in their name.

When used in conjunction with ` + "`qri connect`" + `, list can list a peer's dataset. You
must have ` + "`qri connect`" + ` running in a separate terminal window.`,
		Example: `  # show all of your datasets:
  qri list

  # show datasets with the substring "new" in their name
  qri list new

  # to view the list of your peer's dataset,
  # in one terminal window:
  qri connect

  # in a separate terminal window, to show all of b5's datasets:
  qri list --peer b5`,
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
	cmd.Flags().IntVar(&o.PageSize, "page-size", 25, "page size of results, default 25")
	cmd.Flags().IntVar(&o.Page, "page", 1, "page number results, default 1")
	cmd.Flags().BoolVarP(&o.Published, "published", "p", false, "list only published datasets")
	cmd.Flags().BoolVarP(&o.ShowNumVersions, "num-versions", "n", false, "show number of versions")
	cmd.Flags().StringVar(&o.Peername, "peer", "", "peer whose datasets to list")

	return cmd
}

// ListOptions encapsulates state for the List command
type ListOptions struct {
	ioes.IOStreams

	Format          string
	PageSize        int
	Page            int
	Term            string
	Peername        string
	Published       bool
	ShowNumVersions bool

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ListOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		o.Term = args[0]
	}
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the list command
func (o *ListOptions) Run() (err error) {

	// convert Page and PageSize to Limit and Offset
	page := util.NewPage(o.Page, o.PageSize)

	refs := []repo.DatasetRef{}
	p := &lib.ListParams{
		Term:            o.Term,
		Peername:        o.Peername,
		Limit:           page.Limit(),
		Offset:          page.Offset(),
		Published:       o.Published,
		ShowNumVersions: o.ShowNumVersions,
	}
	if err = o.DatasetRequests.List(p, &refs); err != nil {
		return err
	}

	for _, ref := range refs {
		// remove profileID so names print pretty
		ref.ProfileID = ""
	}

	if len(refs) == 0 {
		if o.Term == "" {
			printInfo(o.Out, "%s has no datasets", o.Peername)
		} else {
			printInfo(o.Out, "%s has no datasets that match \"%s\"", o.Peername, o.Term)
		}
		return
	}

	switch o.Format {
	case "":
		items := make([]fmt.Stringer, len(refs))
		for i, r := range refs {
			items[i] = ref(r)
		}
		printItems(o.Out, items)
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
