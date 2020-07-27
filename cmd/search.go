package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewSearchCommand creates a new `qri search` command that searches for datasets
func NewSearchCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &SearchOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "search QUERY",
		Short: "search the registry for datasets",
		Long: `Search datasets & peers that match your query. Search pings the qri registry. 

Any dataset that has been pushed to the registry is available for search.`,
		Example: `  # Search for datasets featuring "annual population":
  $ qri search "annual population"`,
		Annotations: map[string]string{
			"group": "network",
		},
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json|simple]")
	cmd.Flags().IntVar(&o.PageSize, "page-size", 25, "page size of results, default 25")
	cmd.Flags().IntVar(&o.Page, "page", 1, "page number of results, default 1")

	return cmd
}

// SearchOptions encapsulates state for the search command
type SearchOptions struct {
	ioes.IOStreams

	Query    string
	Format   string
	PageSize int
	Page     int
	// Reindex bool

	SearchMethods *lib.SearchMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *SearchOptions) Complete(f Factory, args []string) (err error) {
	if len(args) != 0 {
		o.Query = args[0]
	}
	o.SearchMethods, err = f.SearchMethods()
	return
}

// Validate checks that any user inputs are valid
func (o *SearchOptions) Validate() error {
	if o.Query == "" {
		return errors.New(lib.ErrBadArgs, "please provide search parameters, for example:\n    $ qri search census\n    $ qri search 'census 2018'\nsee `qri search --help` for more information")
	}
	return nil
}

// Run executes the search command
func (o *SearchOptions) Run() (err error) {
	o.StartSpinner()
	defer o.StopSpinner()

	// TODO: add reindex option back in

	// convert Page and PageSize to Limit and Offset
	page := util.NewPage(o.Page, o.PageSize)

	p := &lib.SearchParams{
		QueryString: o.Query,
		Limit:       page.Limit(),
		Offset:      page.Offset(),
	}

	results := []lib.SearchResult{}

	if err = o.SearchMethods.Search(p, &results); err != nil {
		return err
	}

	// o.StopSpinner()
	switch o.Format {
	case "":
		fmt.Fprintf(o.Out, "showing %d results for '%s'\n", len(results), o.Query)
		items := make([]fmt.Stringer, len(results))
		for i, result := range results {
			items[i] = searchResultStringer(result)
		}
		o.StopSpinner()
		printItems(o.Out, items, page.Offset())
		return nil
	case "simple":
		items := make([]string, len(results))
		for i, r := range results {
			items[i] = fmt.Sprintf("%s/%s", r.Value.Peername, r.Value.Name)
		}
		printlnStringItems(o.Out, items)
		return nil
	case dataset.JSONDataFormat.String():
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		buf := bytes.NewBuffer(data)
		o.StopSpinner()
		printToPager(o.Out, buf)
	default:
		return fmt.Errorf("unrecognized format: %s", o.Format)
	}
	return nil
}
