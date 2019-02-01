package cmd

import (
	"fmt"
	"regexp"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewGetCommand creates a new `qri search` command that searches for datasets
func NewGetCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &GetOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get elements of qri datasets",
		Long: `Get the qri dataset (except for the body). You can also get portions of 
the dataset: meta, structure, viz, transform, and commit. To narrow down
further to specific fields in each section, use dot notation. The get 
command prints to the console in yaml format, by default.

You can get pertinent information on multiple datasets at the same time
by supplying more than one dataset reference.

Check out https://qri.io/docs/reference/dataset/ to learn about each section of the 
dataset and its fields.`,
		Example: `  # print the entire dataset to the console
  qri get me/annual_pop

  # print the meta to the console
  qri get meta me/annual_pop

  # print the dataset body size to the console
  qri get structure.length me/annual_pop

  # print the dataset body size for two different datasets
  qri get structure.length me/annual_pop me/annual_gdp`,
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

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json, yaml]")
	cmd.Flags().BoolVar(&o.Concise, "concise", false, "print output without indentation, only applies to json format")
	cmd.Flags().IntVarP(&o.Limit, "limit", "l", 10, "for body, limit how many entries to get")
	cmd.Flags().IntVarP(&o.Offset, "offset", "s", 0, "for body, offset at which to get entries")
	cmd.Flags().BoolVarP(&o.All, "all", "a", true, "for body, whether to get all entries")

	return cmd
}

// GetOptions encapsulates state for the get command
type GetOptions struct {
	ioes.IOStreams

	Refs     []string
	Selector string
	Format   string
	Concise  bool

	Limit  int
	Offset int
	All    bool

	DatasetRequests *lib.DatasetRequests
}

// isDatasetField checks if a string is a dataset field or not
var isDatasetField = regexp.MustCompile("(?i)^(commit|structure|body|meta|viz|transform)($|\\.)")

// Complete adds any missing configuration that can only be added just before calling Run
func (o *GetOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		if isDatasetField.MatchString(args[0]) {
			o.Selector = args[0]
			args = args[1:]
		}
	}
	o.Refs = args
	o.DatasetRequests, err = f.DatasetRequests()

	if o.Selector != "body" {
		if o.Limit != -1 {
			return fmt.Errorf("can only use --limit flag when getting body")
		}
		if o.Offset != -1 {
			return fmt.Errorf("can only use --offset flag when getting body")
		}
		if !o.All {
			return fmt.Errorf("can only use --all flag when getting body")
		}
	}

	return nil
}

// Run executes the get command
func (o *GetOptions) Run() (err error) {
	var path string
	if len(o.Refs) > 0 {
		path = o.Refs[0]
		if err != nil {
			return err
		}
	}

	p := lib.GetParams{
		Path:     path,
		Selector: o.Selector,
		Format:   o.Format,
		Concise:  o.Concise,
		Offset:   o.Offset,
		Limit:    o.Limit,
		All:      o.All,
	}
	res := lib.GetResult{}
	if err = o.DatasetRequests.Get(&p, &res); err != nil {
		return err
	}

	_, err = o.Out.Write(res.Bytes)
	return err
}
