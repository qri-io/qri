package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

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
	cmd.Flags().IntVar(&o.Offset, "offset", 0, "number of records to skip from results, default 0")
	cmd.Flags().IntVar(&o.Limit, "limit", 25, "size of results, default 25")

	return cmd
}

// SearchOptions encapsulates state for the search command
type SearchOptions struct {
	ioes.IOStreams

	Query  string
	Format string
	Offset int
	Limit  int
	// Reindex bool

	Instance *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *SearchOptions) Complete(f Factory, args []string) (err error) {
	if o.Instance, err = f.Instance(); err != nil {
		return err
	}
	if len(args) != 0 {
		o.Query = args[0]
	}
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
	ctx := context.TODO()
	inst := o.Instance

	o.StartSpinner()
	defer o.StopSpinner()

	// TODO: add reindex option back in

	p := &lib.SearchParams{
		Query:  o.Query,
		Offset: o.Offset,
		Limit:  o.Limit,
	}

	results, err := inst.Search().Search(ctx, p)
	if err != nil {
		return err
	}

	switch o.Format {
	case "":
		fmt.Fprintf(o.Out, "showing %d results for '%s'\n", len(results), o.Query)
		items := make([]fmt.Stringer, len(results))
		for i, result := range results {
			items[i] = searchResultStringer(result)
		}
		o.StopSpinner()
		printItems(o.Out, items, o.Offset)
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
