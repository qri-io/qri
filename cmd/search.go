package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
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

	cmd.Flags().BoolVarP(&o.Reindex, "reindex", "r", false, "re-generate search index from scratch. might take a while.")
	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json]")

	return cmd
}

// SearchOptions encapsulates state for the search command
type SearchOptions struct {
	IOStreams

	Query   string
	Format  string
	Reindex bool

	SearchRequests *core.SearchRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *SearchOptions) Complete(f Factory, args []string) (err error) {
	o.Query = args[0]
	o.SearchRequests, err = f.SearchRequests()
	return
}

// Run executes the search command
func (o *SearchOptions) Run() (err error) {

	if o.Reindex {
		printInfo(o.Out, "building index...")
		done := false
		if err = o.SearchRequests.Reindex(&core.ReindexSearchParams{}, &done); err != nil {
			return err
		}
		printSuccess(o.Out, "reindex complete")
	}

	p := &repo.SearchParams{
		Q:      o.Query,
		Limit:  30,
		Offset: 0,
	}
	res := []repo.DatasetRef{}

	if err = o.SearchRequests.Search(p, &res); err != nil {
		return err
	}

	switch o.Format {
	case "":
		for i, ref := range res {
			printDatasetRefInfo(o.Out, i, ref)
		}
	case dataset.JSONDataFormat.String():
		data, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintf(o.Out, "%s\n", string(data))
	default:
		return fmt.Errorf("unrecognized format: %s", o.Format)
	}
	return nil
}
