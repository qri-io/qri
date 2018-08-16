package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewSearchCommand creates a new `qri search` command that searches for datasets
func NewSearchCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &SearchOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search qri",
		Long:  `Search datasets & peers that match your query. Search pings the qri registry. Any dataset that has been published to the registry is available for search.`,
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
			if err := o.Run(); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json]")

	return cmd
}

// SearchOptions encapsulates state for the search command
type SearchOptions struct {
	IOStreams
	Query          string
	SearchRequests *lib.SearchRequests
	Format         string
	// TODO: add support for specifying limit and offset
	// Limit int
	// Offset int
	// Reindex bool
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *SearchOptions) Complete(f Factory, args []string) (err error) {
	if len(args) != 0 {
		o.Query = args[0]
	}
	o.SearchRequests, err = f.SearchRequests()
	return
}

// Validate checks that any user inputs are valid
func (o *SearchOptions) Validate() error {
	if o.Query == "" {
		return lib.NewError(ErrBadArgs, "please provide search parameters, for example:\n    $ qri search census\n    $ qri search 'census 2018'\nsee `qri search --help` for more information")
	}
	return nil
}

// Run executes the search command
func (o *SearchOptions) Run() (err error) {

	// TODO: add reindex option back in

	p := &lib.SearchParams{
		QueryString: o.Query,
		Limit:       100,
		Offset:      0,
	}

	results := []lib.SearchResult{}

	if err = o.SearchRequests.Search(p, &results); err != nil {
		return err
	}

	switch o.Format {
	case "":
		fmt.Fprintf(o.Out, "showing %d results for '%s'\n", len(results), o.Query)
		for i, result := range results {
			printSearchResult(o.Out, i, result)
		}

	case dataset.JSONDataFormat.String():
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(o.Out, "%s\n", string(data))
	default:
		return fmt.Errorf("unrecognized format: %s", o.Format)
	}
	return nil
}
