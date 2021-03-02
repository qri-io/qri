package cmd

import (
	"context"
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
	cmd.Flags().StringVar(&o.Format, "format", "", "format of dataset body")
	cmd.Flags().StringVar(&o.BodyPath, "body", "", "path to the body file")
	cmd.Flags().BoolVarP(&o.UseDscache, "use-dscache", "", false, "experimental: build and use dscache if none exists")

	return cmd
}

// InitOptions encapsulates state for the `init` command
type InitOptions struct {
	ioes.IOStreams

	Instance *lib.Instance

	// Name of the dataset that will be created
	Name string
	// Format of the body
	Format string
	// Body file to initialize dataset with, if blank an example csv will be used
	BodyPath string
	// Path to use as a working directory. Will be created if it does not exist yet.
	TargetDir string
	// Experimental: Use dscache subsystem to store dataset references
	UseDscache bool

	DatasetMethods *lib.DatasetMethods
}

// Complete completes a dataset reference
func (o *InitOptions) Complete(f Factory, args []string) (err error) {
	if o.DatasetMethods, err = f.DatasetMethods(); err != nil {
		return err
	}
	o.Instance = f.Instance()
	if len(args) > 0 {
		o.TargetDir = args[0]
	}
	return err
}

// Run executes the `init` command
func (o *InitOptions) Run() (err error) {
	ctx := context.TODO()
	inst := o.Instance

	// An empty dir means the current dir
	if o.TargetDir == "" {
		o.TargetDir, _ = os.Getwd()
	}

	// First, check if the directory can be init'd, before prompting for any input
	canInitParams := lib.InitDatasetParams{
		TargetDir: o.TargetDir,
		BodyPath:  o.BodyPath,
	}
	if err = inst.Filesys().CanInitDatasetWorkDir(ctx, &canInitParams); err != nil {
		return err
	}

	// Suggestion for the dataset name defaults to directory it is being linked into
	if o.Name == "" {
		suggestedName := dsref.GenerateName(filepath.Base(o.TargetDir), "dataset_")
		o.Name = inputText(o.Out, o.In, "Name of new dataset", suggestedName)
	}

	// If user inputted their own dataset name, make sure it's valid.
	if err := dsref.EnsureValidName(o.Name); err != nil {
		return err
	}

	// If --body flag is set, use that to figure out the format
	if o.BodyPath != "" {
		ext := filepath.Ext(o.BodyPath)
		if strings.HasPrefix(ext, ".") {
			ext = ext[1:]
		}
		o.Format = ext
	}

	if o.Format == "" {
		o.Format = inputText(o.Out, o.In, "Format of dataset, csv or json", "csv")
	}

	p := &lib.InitDatasetParams{
		TargetDir:  o.TargetDir,
		Format:     o.Format,
		Name:       o.Name,
		BodyPath:   o.BodyPath,
		UseDscache: o.UseDscache,
	}

	refstr, err := inst.Filesys().Init(ctx, p)
	if err != nil {
		return err
	}

	printSuccess(o.Out, "initialized working directory for new dataset %s", refstr)
	return nil
}
