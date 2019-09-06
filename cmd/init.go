package cmd

import (
	"os"
	"path/filepath"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/varName"
	"github.com/spf13/cobra"
)

// NewInitCommand creates new `qri init` command that connects a working directory in
// the local filesystem to a dataset your repo.
func NewInitCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &InitOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "initialize a dataset directory",
		Long:    ``,
		Example: ``,
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

	cmd.Flags().StringVar(&o.Name, "name", "", "name of the dataset")
	cmd.Flags().StringVar(&o.Format, "format", "", "format of dataset")
	cmd.Flags().StringVar(&o.SourceBodyPath, "source-body-path", "", "path to the body file")

	return cmd
}

// InitOptions encapsulates state for the `init` command
type InitOptions struct {
	ioes.IOStreams

	Name           string
	Format         string
	SourceBodyPath string
	Mkdir          string

	DatasetRequests *lib.DatasetRequests
	FSIMethods      *lib.FSIMethods
}

// Complete completes a dataset reference
func (o *InitOptions) Complete(f Factory, args []string) (err error) {
	if o.DatasetRequests, err = f.DatasetRequests(); err != nil {
		return err
	}
	o.FSIMethods, err = f.FSIMethods()
	if len(args) > 0 && args[0] != "." {
		o.Mkdir = args[0]
	}
	return err
}

// Run executes the `init` command
func (o *InitOptions) Run() (err error) {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Suggestion for the dataset name defaults to directory it is being linked into
	if o.Name == "" {
		var suggestedName string
		if o.Mkdir == "" {
			suggestedName = varName.CreateVarNameFromString(filepath.Base(pwd))
		} else {
			suggestedName = varName.CreateVarNameFromString(o.Mkdir)
		}
		o.Name = inputText(o.ErrOut, o.In, "Name of new dataset", suggestedName)
	}

	if o.Format == "" {
		o.Format = inputText(o.ErrOut, o.In, "Format of dataset, csv or json", "csv")
	}

	p := &lib.InitFSIDatasetParams{
		Dir:            pwd,
		Mkdir:          o.Mkdir,
		Format:         o.Format,
		Name:           o.Name,
		SourceBodyPath: o.SourceBodyPath,
	}
	var name string
	if err = o.FSIMethods.InitDataset(p, &name); err != nil {
		return err
	}

	printSuccess(o.Out, "initialized working directory for new dataset %s", name)
	return nil
}
