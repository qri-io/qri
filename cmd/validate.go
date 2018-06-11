package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewValidateCommand creates a new `qri validate` cobra command for showing schema errors
// in a dataset body
func NewValidateCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &ValidateOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "show schema validation errors",
		Annotations: map[string]string{
			"group": "dataset",
		},
		Long: `
Validate checks data for errors using a schema and then printing a list of 
issues. By default validate checks dataset data against it’s own schema. 
Validate is a flexible command that works with data and schemas either 
inside or outside of qri by providing one or both of --data and --schema 
arguments. 

Providing --schema and --data is an “external validation" that uses nothing 
stored in qri. When only one of schema or data args are provided, the other 
comes from a dataset reference. For example, to check how a file “data.csv” 
validates against a dataset "foo”, we would run:

  $ qri validate --data data.csv me/foo

In this case, qri will will print any validation as if data.csv was foo’s data.

To see how changes to a schema will validate against a 
dataset in qri, we would run:

  $ qri validate --schema schema.json me/foo

In this case, qri will print validation errors as if stucture.json was the
schema for dataset "me/foo"

Using validate this way is a great way to see how changes to data or schema
will affect a dataset before saving changes to a dataset.`,
		Example: `  show errors in an existing dataset:
  $ qri validate b5/comics`,
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Run())
		},
	}

	cmd.Flags().StringVarP(&o.URL, "url", "u", "", "url to file to initialize from")
	cmd.Flags().StringVarP(&o.Filepath, "data", "f", "", "data file to initialize from")
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

	DatasetRequests *core.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ValidateOptions) Complete(f Factory, args []string) (err error) {
	if f.RPC() != nil {
		return usingRPCError("validate")
	}

	o.Ref = args[0]
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the validate command
func (o *ValidateOptions) Run() (err error) {
	var (
		dataFile, schemaFile *os.File
		ref                  repo.DatasetRef
	)

	if o.Ref != "" {
		ref, err = repo.ParseDatasetRef(o.Ref)
		if err != nil {
			return err
		}
	}

	if ref.IsEmpty() && !(o.Filepath != "" && o.SchemaFilepath != "") {
		ErrExit(fmt.Errorf("please provide a dataset name to validate, or both  --data and --schema arguments"))
	}

	if dataFile, err = loadFileIfPath(o.Filepath); err != nil {
		return err
	}
	if schemaFile, err = loadFileIfPath(o.SchemaFilepath); err != nil {
		return err
	}

	p := &core.ValidateDatasetParams{
		Ref: ref,
		// URL:          addDsURL,
		DataFilename: filepath.Base(o.SchemaFilepath),
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
		return err
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
