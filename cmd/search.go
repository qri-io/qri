package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
)

// NewSearchCommand creates a new `qri search` command that searches for datasets
func NewSearchCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &SearchOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search qri",
		Long:  `Search datasets & peers that match your query`,
		Annotations: map[string]string{
			"group": "other",
		},
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Run())
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json]")

	return cmd
}

// SearchOptions encapsulates state for the search command
type SearchOptions struct {
	IOStreams
	Query          string
	SearchRequests *core.SearchRequests
	Format         string
	// TODO: add support for specifying limit and offset
	// Limit int
	// Offset int
	// Reindex bool
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *SearchOptions) Complete(f Factory, args []string) (err error) {
	o.Query = args[0]
	o.SearchRequests, err = f.SearchRequests()
	return
}

// Run executes the search command
func (o *SearchOptions) Run() (err error) {

	// TODO: add reindex option back in

	p := &core.SearchParams{
		QueryString: o.Query,
		Limit:       100,
		Offset:      0,
	}
	results := &[]core.SearchResult{}

	if err = o.SearchRequests.Search(p, results); err != nil {
		return err
	}

	switch o.Format {
	case "":
		fmt.Printf("showing %d results for '%s'\n", len(*results), o.Query)
		for i, result := range *results {
			printSearchResult(i, result)
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
