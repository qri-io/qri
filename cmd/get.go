package cmd

import (
	"fmt"
	"regexp"

	"encoding/json"

	"github.com/ghodss/yaml"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
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

	cmd.Flags().StringVarP(&o.Format, "format", "f", "yaml", "set output format [json, yaml]")
	cmd.Flags().BoolVar(&o.Concise, "concise", false, "print output without indentation, only applies to json format")

	return cmd
}

// GetOptions encapsulates state for the get command
type GetOptions struct {
	ioes.IOStreams

	Refs    []string
	Path    string
	Format  string
	Concise bool

	DatasetRequests *lib.DatasetRequests
}

// isDatasetField checks if a string is a dataset field or not
var isDatasetField = regexp.MustCompile("(?i)^(commit|structure|body|meta|viz|transform)($|\\.)")

// Complete adds any missing configuration that can only be added just before calling Run
func (o *GetOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		if isDatasetField.MatchString(args[0]) {
			o.Path = args[0]
			args = args[1:]
		}
	}
	o.Refs = args
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the get command
func (o *GetOptions) Run() (err error) {
	var ref repo.DatasetRef
	if len(o.Refs) > 0 {
		ref, err = repo.ParseDatasetRef(o.Refs[0])
		if err != nil {
			return err
		}
	}

	// TODO: It is more efficient to only request data in the Path field, but for now
	// just post-process the less efficient full lookup.
	res := repo.DatasetRef{}
	if err = o.DatasetRequests.Get(&ref, &res); err != nil {
		return err
	}

	// TOOD: Specially handle `body` to call LookupBody on the dataset.
	var value interface{}
	if o.Path == "" {
		value = res
	} else {
		// TODO: Don't depend directly on base.
		value, err = base.ApplyPath(res.Dataset, o.Path)
		if err != nil {
			return err
		}
	}

	encode := map[string]interface{}{}
	encode[res.String()] = value

	var buffer []byte
	switch o.Format {
	case "json":
		if o.Concise {
			buffer, err = json.Marshal(encode)
		} else {
			buffer, err = json.MarshalIndent(encode, "", " ")
		}
	case "yaml":
		buffer, err = yaml.Marshal(encode)
	}
	if err != nil {
		return fmt.Errorf("error getting config: %s", err)
	}
	_, err = o.Out.Write(buffer)
	return err
}
