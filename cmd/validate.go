package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewValidateCommand creates a new `qri validate` cobra command for showing schema errors
// in a dataset body
func NewValidateCommand(f Factory, ioStreams IOStreams) *cobra.Command {
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
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	// TODO: restore
	// cmd.Flags().StringVarP(&o.URL, "url", "u", "", "url to file to initialize from")
	cmd.Flags().StringVarP(&o.Filepath, "body", "b", "", "data file to initialize from")
	cmd.Flags().StringVarP(&o.SchemaFilepath, "schema", "", "", "json schema file to use for validation")

	return cmd
}

// ValidateOptions encapsulates state for the validate command
type ValidateOptions struct {
	IOStreams

	Ref            string
	Filepath       string
	SchemaFilepath string
	URL            string
	// validateDsPassive        bool

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any configuration that can only be added just before calling Run
func (o *ValidateOptions) Complete(f Factory, args []string) (err error) {
	if f.RPC() != nil {
		return usingRPCError("validate")
	}

	if len(args) != 0 {
		o.Ref = args[0]
	}

	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Validate checks that any user inputs are valid
func (o *ValidateOptions) Validate() error {
	if o.URL != "" && o.Ref == "" && o.SchemaFilepath == "" {
		return (lib.NewError(ErrBadArgs, "if you are validating data from a url, please include a dataset name or supply the --schema flag with a file path that Qri can validate against"))
	}
	if o.Ref == "" && o.Filepath == "" && o.SchemaFilepath == "" {
		return lib.NewError(ErrBadArgs, "please provide a dataset name, or a supply the --body and --schema flags with file paths")
	}
	return nil
}

// Run executes the run command
func (o *ValidateOptions) Run() (err error) {
	var (
		dataFile, schemaFile *os.File
		ref                  repo.DatasetRef
	)

	if o.Ref != "" {
		ref, err = repo.ParseDatasetRef(o.Ref)
		if err != nil {
			return lib.NewError(err, fmt.Sprintf("%s must be in correct DatasetRef format, [peername]/[datatset_name]", o.Ref))
		}
	}

	if dataFile, err = loadFileIfPath(o.Filepath); err != nil {
		return lib.NewError(err, fmt.Sprintf("error opening body file: could not %s", err))
	}
	if schemaFile, err = loadFileIfPath(o.SchemaFilepath); err != nil {
		return lib.NewError(err, fmt.Sprintf("error opening schema file: could not %s", err))
	}

	p := &lib.ValidateDatasetParams{
		Ref: ref,
		// TODO: restore
		// URL:          addDsURL,
		DataFilename: filepath.Base(o.Filepath),
	}

	// this is because passing nil to interfaces is bad
	// see: https://golang.org/doc/faq#nil_error
	if dataFile != nil {
		p.Data = dataFile
	}
	if schemaFile != nil {
		p.Schema = schemaFile
	}

	res := []jsonschema.ValError{}
	if err = o.DatasetRequests.Validate(p, &res); err != nil {
		return lib.NewError(err, "")
	}
	if len(res) == 0 {
		printSuccess(o.Out, "✔ All good!")
		return
	}

	for i, err := range res {
		fmt.Fprintf(o.Out, "%d: %s\n", i, err.Error())
	}
	return nil
}
