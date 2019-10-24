package cmd

import (
	"encoding/json"
	"os"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/fsi/component"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewDiffCommand creates a new `qri diff` cobra command for comparing changes between datasets
func NewDiffCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := DiffOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare differences between two data sources",
		Long: `diff is a new & experimental feature, please report bugs here:
https://github.com/qri-io/deepdiff

Diff compares two data sources & generates a description of the difference
between them. The output of diff describes the steps required to make the 
element on the left (the first argument) equal the element on the right (the
second argument). The steps themselves are the "diff".

Unlike the classic unix diff utility (which operates on text),
qri diff works on structured data. qri diffs are measured in elements
(think cells in a spreadsheet), each change is either an insert (added 
elements), delete (removed elements), or update (changed values).

Each change has a path that locates it within the document`,
		Example: `  diff between a latest version & the next one back:
  $ qri diff me/annual_pop

  diff current "qri use" selection:
  $ qri diff

  diff dataset body against it's last version
  $ qri diff body me/annual_pop

  diff two dataset meta sections:
  $ qri diff meta me/population_2016 me/population_2017

  diff two local json files:
  $ qri diff a.json b.json

  diff a json & csv file
  $ qri diff some_table.csv b.json`,
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

	cmd.Flags().StringVarP(&o.Format, "format", "f", "pretty", "output format. one of [json,pretty]")
	cmd.Flags().BoolVar(&o.Summary, "summary", false, "just output the summary")

	return cmd
}

// DiffOptions encapsulates options for the diff command
type DiffOptions struct {
	ioes.IOStreams

	Refs     *RefSelect
	Selector string
	Format   string
	Summary  bool

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *DiffOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		if component.IsDatasetField.MatchString(args[0]) {
			o.Selector = args[0]
			args = args[1:]
		}
	}
	o.DatasetRequests, err = f.DatasetRequests()

	if o.Refs, err = GetCurrentRefSelect(f, args, 2); err != nil {
		return
	}
	return
}

// Run executes the diff command
func (o *DiffOptions) Run() (err error) {
	printRefSelect(o.Out, o.Refs)

	p := &lib.DiffParams{
		Selector: o.Selector,
	}

	if o.Refs.IsLinked() {
		return o.RunLinkedFilesys(p)
	} else if len(o.Refs.RefList()) == 1 {
		p.RightPath = o.Refs.Ref()
	} else if len(o.Refs.RefList()) == 2 {
		p.LeftPath = o.Refs.RefList()[0]
		p.RightPath = o.Refs.RefList()[1]
	}

	res := &lib.DiffResponse{}
	if err = o.DatasetRequests.Diff(p, res); err != nil {
		return err
	}

	if o.Format == "json" {
		json.NewEncoder(o.Out).Encode(res.Diff)
		return
	}

	return printDiff(o.Out, res, o.Summary)
}

// RunLinkedFilesys executes diff against a linked directory
func (o *DiffOptions) RunLinkedFilesys(p *lib.DiffParams) (err error) {
	responses := []lib.DiffResponse{}
	components := []string{}

	// TODO(dlong): Move to FSI.

	// Meta component.
	p.LeftPath = o.Refs.Ref()
	p.RightPath = "meta.json"
	p.Selector = "meta"
	res := lib.DiffResponse{}
	if err = o.DatasetRequests.Diff(p, &res); err != nil {
		return err
	}
	responses = append(responses, res)
	components = append(components, "meta")

	// Body component.
	if _, err = os.Stat("body.csv"); !os.IsNotExist(err) {
		p.RightPath = "body.csv"
	}
	if _, err = os.Stat("body.json"); !os.IsNotExist(err) {
		p.RightPath = "body.json"
	}
	p.Selector = "body"
	res = lib.DiffResponse{}
	if err = o.DatasetRequests.Diff(p, &res); err != nil {
		return err
	}
	responses = append(responses, res)
	components = append(components, "body")

	// Schema component.
	p.LeftPath = o.Refs.Ref()
	p.RightPath = "schema.json"
	p.Selector = "structure.schema"
	res = lib.DiffResponse{}
	if err = o.DatasetRequests.Diff(p, &res); err != nil {
		return err
	}
	responses = append(responses, res)
	components = append(components, "schema")

	mergedResponse := lib.DiffResponse{}
	if err = o.DatasetRequests.MergeDiffs(&mergedResponse, responses, components); err != nil {
		return err
	}
	return printDiff(o.Out, &mergedResponse, o.Summary)
}
