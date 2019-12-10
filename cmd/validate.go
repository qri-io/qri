package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/ioes"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewValidateCommand creates a new `qri validate` cobra command for showing schema errors
// in a dataset body
func NewValidateCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ValidateOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Show schema validation errors",
		Annotations: map[string]string{
			"group": "dataset",
		},
		Long: `
Validate checks data for errors using a schema and then printing a list of 
issues. By default validate checks a dataset's body against it’s own schema. 
Validate is a flexible command that works with data and schemas either 
inside or outside of qri by providing one or both of --body and --schema 
arguments. 

Providing --schema and --body is an “external validation" that uses nothing 
stored in qri. When only one of schema or body args are provided, the other 
comes from a dataset reference. For example, to check how a file “data.csv” 
validates against a dataset "foo”, we would run:

  $ qri validate --body data.csv me/foo

In this case, qri will will print any validation as if data.csv was foo’s data.

To see how changes to a schema will validate against a 
dataset in qri, we would run:

  $ qri validate --schema schema.json me/foo

In this case, qri will print validation errors as if stucture.json was the
schema for dataset "me/foo"

Using validate this way is a great way to see how changes to data or schema
will affect a dataset before saving changes to a dataset.

You can get the current schema of a dataset by running the ` + "`qri get structure.schema`" + `
command.

Note: --body and --schema flags will override the dataset if both flags are provided.`,
		Example: `  # show errors in an existing dataset:
  qri validate b5/comics

  # validate a new body against an existing schema
  qri validate --body new_data.csv me/annual_pop

  # validate data against a new schema
  qri validate --body data.csv --schema schema.json`,
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
	cmd.Flags().StringVarP(&o.SchemaFilepath, "schema", "", "", "json schema file to use for validation")

	return cmd
}

// ValidateOptions encapsulates state for the validate command
type ValidateOptions struct {
	ioes.IOStreams

	Refs           *RefSelect
	BodyFilepath   string
	SchemaFilepath string
	URL            string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any configuration that can only be added just before calling Run
func (o *ValidateOptions) Complete(f Factory, args []string) (err error) {
	if o.DatasetRequests, err = f.DatasetRequests(); err != nil {
		return
	}

	o.Refs, err = GetCurrentRefSelect(f, args, 1, nil)
	if err == repo.ErrEmptyRef {
		// It is not an error to call validate without a dataset reference. Might be
		// validating a body file against a schema file directly.
		o.Refs = NewEmptyRefSelect()
		err = nil
	}
	return
}

// Run executes the run command
func (o *ValidateOptions) Run() (err error) {
	var (
		bodyFile, schemaFile *os.File
	)

	printRefSelect(o.Out, o.Refs)

	o.StartSpinner()
	defer o.StopSpinner()

	if o.Refs.IsLinked() {
		if o.BodyFilepath == "" {
			// TODO(dlong): FSI should determine the filename by looking for each known file
			// extension.
			if _, err := os.Stat("body.json"); !os.IsNotExist(err) {
				o.BodyFilepath = "body.json"
			}
			if _, err := os.Stat("body.csv"); !os.IsNotExist(err) {
				o.BodyFilepath = "body.csv"
			}
			if err = qfs.AbsPath(&o.BodyFilepath); err != nil {
				return err
			}
		}
		if o.SchemaFilepath == "" {
			o.SchemaFilepath = "schema.json"
			if err = qfs.AbsPath(&o.SchemaFilepath); err != nil {
				return err
			}
		}
	}

	if o.BodyFilepath != "" {
		if bodyFile, err = loadFileIfPath(o.BodyFilepath); err != nil {
			return lib.NewError(err, fmt.Sprintf("error opening body file: could not %s", err))
		}
	}
	if o.SchemaFilepath != "" {
		if schemaFile, err = loadFileIfPath(o.SchemaFilepath); err != nil {
			return lib.NewError(err, fmt.Sprintf("error opening schema file: could not %s", err))
		}
	}

	ref := o.Refs.Ref()
	p := &lib.ValidateDatasetParams{
		Ref: ref,
		// TODO: restore
		// URL:          addDsURL,
		BodyFilename: filepath.Base(o.BodyFilepath),
	}

	// this is because passing nil to interfaces is bad
	// see: https://golang.org/doc/faq#nil_error
	if bodyFile != nil {
		p.Body = bodyFile
	}
	if schemaFile != nil {
		p.Schema = schemaFile
	}

	res := []jsonschema.ValError{}
	if err = o.DatasetRequests.Validate(p, &res); err != nil {
		return err
	}

	o.StopSpinner()

	if len(res) == 0 {
		printSuccess(o.Out, "✔ All good!")
		return
	}

	for i, err := range res {
		fmt.Fprintf(o.Out, "%d: %s\n", i, err.Error())
	}
	return nil
}
