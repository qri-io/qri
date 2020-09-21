package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewInitCommand creates new `qri init` command that connects a working directory in
// the local filesystem to a dataset your repo.
func NewInitCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &InitOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "init [PATH]",
		Short: "initialize a dataset directory",
		Long: `'initialize' creates a new dataset, links it to the current directory,
and creates starter files for the dataset's components. You can also specify an
already existing body file using the 'body' flag.`,
		Example: `  # initialize a new dataset, linking it to the current directory:

  $ mkdir earthquakes && cd earthquakes
  $ qri init
  Name of new dataset [earthquakes]: 
  Format of dataset, csv or json [csv]: csv
  initialized working directory for new dataset user/earthquakes
	
	# initialize a new dataset, specifying a file to use as the body of the dataset:
	
  $ qri init --body /Users/datasets/earthquakes.csv --name earthquakes
  initialized working directory for new dataset user/earthquakes
`,
		Annotations: map[string]string{
			"group": "workdir",
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
	cmd.Flags().StringVar(&o.SourceBodyPath, "body", "", "path to the body file")
	cmd.Flags().BoolVarP(&o.UseDscache, "use-dscache", "", false, "experimental: build and use dscache if none exists")

	return cmd
}

// InitOptions encapsulates state for the `init` command
type InitOptions struct {
	ioes.IOStreams

	Name           string
	Format         string
	SourceBodyPath string
	Mkdir          string
	UseDscache     bool

	DatasetMethods *lib.DatasetMethods
	FSIMethods     *lib.FSIMethods
}

// Complete completes a dataset reference
func (o *InitOptions) Complete(f Factory, args []string) (err error) {
	if o.DatasetMethods, err = f.DatasetMethods(); err != nil {
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
	targetDir := pwd
	if o.Mkdir != "" {
		targetDir = o.Mkdir
	}

	// First, check if the directory can be init'd, before prompting for any input
	canInitParams := lib.InitFSIDatasetParams{
		Dir:            targetDir,
		SourceBodyPath: o.SourceBodyPath,
	}
	if err = o.FSIMethods.CanInitDatasetWorkDir(&canInitParams, nil); err != nil {
		return err
	}

	// Suggestion for the dataset name defaults to directory it is being linked into
	if o.Name == "" {
		// TODO(dustmop): Currently all tests that call `init` use the --name flag. Add a test
		// that receives stdin and checks what is written to stdout.
		suggestedName := dsref.GenerateName(filepath.Base(targetDir), "dataset_")
		o.Name = inputText(o.Out, o.In, "Name of new dataset", suggestedName)
	}

	// If user inputted there own dataset name, make sure it's valid.
	if err := dsref.EnsureValidName(o.Name); err != nil {
		return err
	}

	// If --body flag is set, use that to figure out the format
	if o.SourceBodyPath != "" {
		ext := filepath.Ext(o.SourceBodyPath)
		if strings.HasPrefix(ext, ".") {
			ext = ext[1:]
		}
		o.Format = ext
	}

	if o.Format == "" {
		o.Format = inputText(o.Out, o.In, "Format of dataset, csv or json", "csv")
	}

	p := &lib.InitFSIDatasetParams{
		Dir:            pwd,
		Mkdir:          o.Mkdir,
		Format:         o.Format,
		Name:           o.Name,
		SourceBodyPath: o.SourceBodyPath,
		UseDscache:     o.UseDscache,
	}
	var refstr string
	if err = o.FSIMethods.InitDataset(p, &refstr); err != nil {
		return err
	}

	printSuccess(o.Out, "initialized working directory for new dataset %s", refstr)
	return nil
}
