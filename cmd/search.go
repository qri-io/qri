package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewSearchCommand creates a new `qri search` command that searches for datasets
func NewSearchCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &SearchOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search qri",
		Long: `
Search datasets & peers that match your query. Search pings the qri registry. 

Any dataset that has been published to the registry is available for search.`,
		Example: `
  # search 
  $ qri search "annual population"`,
		Annotations: map[string]string{
			"group": "network",
		},
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

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json]")
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
		return lib.NewError(lib.ErrBadArgs, "please provide search parameters, for example:\n    $ qri search census\n    $ qri search 'census 2018'\nsee `qri search --help` for more information")
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
			ref, err := searchResultToRef(&result)
			if err != nil {
				return err
			}
			items[i] = refStringer(*ref)
		}
		o.StopSpinner()
		printItems(o.Out, items, page.Offset())
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

func searchResultToRef(result *lib.SearchResult) (*repo.DatasetRef, error) {
	ref := &repo.DatasetRef{
		Dataset: &dataset.Dataset{},
	}
	raw, err := json.Marshal(result.Value)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, ref.Dataset); err != nil {
		return nil, err
	}
	ref.Path = ref.Dataset.Path

	id := strings.Split(result.ID, "/")
	if len(id) != 2 {
		ref.Peername = ref.Dataset.Peername
		ref.Name = ref.Dataset.Name
		return ref, nil
	}
	ref.Peername = id[0]
	ref.Name = id[1]
	return ref, nil
}
