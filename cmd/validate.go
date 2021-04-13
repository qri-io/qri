package cmd

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewValidateCommand creates a new `qri validate` cobra command for showing schema errors
// in a dataset body
func NewValidateCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ValidateOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "validate [DATASET]",
		Short: "show schema validation errors",
		Annotations: map[string]string{
			"group": "dataset",
		},
		Long: `Validate checks data for errors using a schema and then printing a list of
issues. By default validate checks a dataset's body against it’s own schema.
Validate is a flexible command that works with data and schemas either
inside or outside of qri by providing the --body and --schema or --structure
flags.

Providing either --schema or --structure and --body is an “external
validation" that uses nothing stored in qri. When only one of these flags,
are provided, the other comes from a dataset reference. For example, to
check how a file “data.csv” validates against a dataset "foo”, we would run:

  $ qri validate --body data.csv me/foo

In this case, qri will will print any validation as if data.csv was foo’s data.

To see how changes to a schema will validate against a dataset in qri, we
would run:

  $ qri validate --schema schema.json me/foo

In this case, qri will print validation errors as if schema.json was the
schema for dataset "me/foo"

Using validate this way is a great way to see how changes to data or schema
will affect a dataset before saving changes to a dataset.

You can get the current schema of a dataset by running the ` + "`qri get structure.schema`" + `
command.

Note: --body and --schema or --structure flags will override the dataset
if these flags are provided.`,
		Example: `  # Show errors in an existing dataset:
  $ qri validate b5/comics

  # Validate a new body against an existing schema:
  $ qri validate --body new_data.csv me/annual_pop

  # Validate data against a new schema:
  $ qri validate --body data.csv --schema schema.json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	// TODO: restore
	// cmd.Flags().StringVarP(&o.URL, "url", "u", "", "url to file to initialize from")
	cmd.Flags().StringVarP(&o.BodyFilepath, "body", "b", "", "body file to validate")
	cmd.MarkFlagFilename("body")
	cmd.Flags().StringVar(&o.SchemaFilepath, "schema", "", "json schema file to use for validation")
	cmd.MarkFlagFilename("schema", "json")
	cmd.Flags().StringVarP(&o.StructureFilepath, "structure", "", "", "json structure file to use for validation")
	cmd.MarkFlagFilename("structure", "json")
	cmd.Flags().StringVar(&o.Format, "format", "table", "output format. One of: [table|json|csv]")

	return cmd
}

// ValidateOptions encapsulates state for the validate command
type ValidateOptions struct {
	ioes.IOStreams

	Refs              *RefSelect
	BodyFilepath      string
	SchemaFilepath    string
	StructureFilepath string
	Format            string

	inst *lib.Instance
}

// Complete adds any configuration that can only be added just before calling Run
func (o *ValidateOptions) Complete(f Factory, args []string) (err error) {
	if o.inst, err = f.Instance(); err != nil {
		return
	}

	if o.Format != "table" && o.Format != "json" && o.Format != "csv" {
		return fmt.Errorf(`%q is not a valid output format. Please use one of: "table", "json", "csv"`, o.Format)
	}

	o.Refs, err = GetCurrentRefSelect(f, args, 1, nil)
	if errors.Is(err, repo.ErrEmptyRef) {
		// It is not an error to call validate without a dataset reference. Might be
		// validating a body file against a schema file directly.
		o.Refs = NewEmptyRefSelect()
		err = nil
	}
	return
}

// Run executes the run command
func (o *ValidateOptions) Run() (err error) {
	printRefSelect(o.ErrOut, o.Refs)

	o.StartSpinner()
	defer o.StopSpinner()

	ref := o.Refs.Ref()
	p := &lib.ValidateParams{
		Ref:               ref,
		BodyFilename:      o.BodyFilepath,
		SchemaFilename:    o.SchemaFilepath,
		StructureFilename: o.StructureFilepath,
	}

	ctx := context.TODO()
	res, err := o.inst.WithSource("local").Dataset().Validate(ctx, p)
	if err != nil {
		return err
	}

	o.StopSpinner()

	switch o.Format {
	case "table":
		if len(res.Errors) == 0 {
			printSuccess(o.Out, "✔ All good!")
			return nil
		}
		header, data := tabularValidationData(res.Structure, res.Errors)
		buf := &bytes.Buffer{}
		renderTable(buf, header, data)
		printToPager(o.Out, buf)
	case "csv":
		header, data := tabularValidationData(res.Structure, res.Errors)
		csv.NewWriter(o.Out).WriteAll(append([][]string{header}, data...))
	case "json":
		if err := json.NewEncoder(o.Out).Encode(res.Errors); err != nil {
			return err
		}
	}
	return nil
}

func tabularValidationData(st *dataset.Structure, errs []jsonschema.KeyError) ([]string, [][]string) {
	var (
		header []string
		data   = make([][]string, len(errs))
	)

	if st.Depth == 2 {
		header = []string{"#", "row", "col", "value", "error"}
		for i, e := range errs {
			paths := strings.Split(e.PropertyPath, "/")
			if len(paths) < 3 {
				paths = []string{"", "", ""}
			}
			data[i] = []string{strconv.FormatInt(int64(i), 10), paths[1], paths[2], valStr(e.InvalidValue), e.Message}
		}
	} else {
		header = []string{"#", "path", "value", "error"}
		for i, e := range errs {
			data[i] = []string{strconv.FormatInt(int64(i), 10), e.PropertyPath, valStr(e.InvalidValue), e.Message}
		}
	}

	return header, data
}

func valStr(v interface{}) string {
	switch x := v.(type) {
	case string:
		if len(x) > 20 {
			x = x[:17] + "..."
		}
		return x
	case int:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'E', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	case nil:
		return "NULL"
	default:
		return "<unknown>"
	}
}
