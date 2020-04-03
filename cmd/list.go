package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewListCommand creates new `qri list` command that lists datasets for the local peer & others
func NewListCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ListOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "list [FILTER]",
		Aliases: []string{"ls"},
		Short:   "show a list of datasets",
		Long: `List shows lists of datasets, including names and current hashes. 

The default list is the latest version of all datasets you have on your local 
qri repository. The first argument can be used to find datasets with a certain 
substring in their name.

When used in conjunction with ` + "`qri connect`" + `, list can list a peer's dataset. You
must have ` + "`qri connect`" + ` running in a separate terminal window.`,
		Example: `  # Show all of your datasets:
  $ qri list

  # Show datasets with the substring "new" in their name:
  $ qri list new

  # To view the list of a peer's datasets...
  # In one terminal window:
  $ qri connect
  # In a separate terminal window, show all of b5's datasets:
  $ qri list --peer b5`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json|simple]")
	cmd.Flags().IntVar(&o.PageSize, "page-size", 25, "page size of results, default 25")
	cmd.Flags().IntVar(&o.Page, "page", 1, "page number results, default 1")
	cmd.Flags().BoolVarP(&o.Published, "published", "p", false, "list only published datasets")
	cmd.Flags().BoolVarP(&o.ShowNumVersions, "num-versions", "n", false, "show number of versions")
	cmd.Flags().StringVar(&o.Peername, "peer", "", "peer whose datasets to list")
	cmd.Flags().BoolVarP(&o.Raw, "raw", "r", false, "to show raw references")
	cmd.Flags().BoolVarP(&o.UseDscache, "use-dscache", "", false, "experimental: build and use dscache to list")

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
	Raw             bool
	UseDscache      bool

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

	if o.Raw {
		var text string
		p := &lib.ListParams{
			UseDscache: o.UseDscache,
		}
		if err = o.DatasetRequests.ListRawRefs(p, &text); err != nil {
			return err
		}
		printSuccess(o.Out, text)
		return nil
	}

	infos := []dsref.VersionInfo{}
	p := &lib.ListParams{
		Term:            o.Term,
		Peername:        o.Peername,
		Limit:           page.Limit(),
		Offset:          page.Offset(),
		Published:       o.Published,
		ShowNumVersions: o.ShowNumVersions,
		EnsureFSIExists: true,
		UseDscache:      o.UseDscache,
	}
	if err = o.DatasetRequests.List(p, &infos); err != nil {
		return err
	}

	for _, ref := range infos {
		// remove profileID so names print pretty
		ref.ProfileID = ""
	}

	if len(infos) == 0 {
		pn := fmt.Sprintf("%s has", o.Peername)
		if o.Peername == "" {
			pn = "you have"
		}

		if o.Term == "" {
			printInfo(o.Out, "%s no datasets", pn)
		} else {
			printInfo(o.Out, "%s no datasets that match \"%s\"", pn, o.Term)
		}
		return
	}

	switch o.Format {
	case "":
		items := make([]fmt.Stringer, len(infos))
		for i, r := range infos {
			items[i] = versionInfoStringer(r)
		}
		printItems(o.Out, items, page.Offset())
		return nil
	case "simple":
		items := make([]string, len(infos))
		for i, r := range infos {
			items[i] = r.SimpleRef().Alias()
		}
		printlnStringItems(o.Out, items)
		return nil
	case dataset.JSONDataFormat.String():
		data, err := json.MarshalIndent(infos, "", "  ")
		if err != nil {
			return err
		}
		buf := bytes.NewBuffer(data)
		printToPager(o.Out, buf)
		return nil
	default:
		return fmt.Errorf("unrecognized format: %s", o.Format)
	}
}
