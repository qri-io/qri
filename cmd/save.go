package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
	"github.com/spf13/cobra"
)

// NewSaveCommand creates a `qri save` cobra command used for saving changes
// to datasets
func NewSaveCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &SaveOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "save",
		Aliases: []string{"commit"},
		Short:   "Save changes to a dataset",
		Long: `
Save is how you change a dataset, updating one or more of data, metadata, and structure. 
You can also update your data via url. Every time you run save, an entry is added to 
your dataset’s log (which you can see by running ` + "`qri log <dataset_reference>`" + `).

If the dataset you're changing has defined a transform, running ` + "`qri save`" + `
will re execute the transform. To only re-run the transform, run save with no args.

Every time you save, you can provide a message about what you changed and why. 
If you don’t provide a message Qri will automatically generate one for you.

When you make an update and save a dataset that you originally added from a different
peer, the dataset gets renamed from ` + "`peers_name/dataset_name`" + ` to ` + "`my_name/dataset_name`" + `.

The ` + "`--message`" + `" and ` + "`--title`" + ` flags allow you to add a 
commit message and title to the save.`,
		Example: `  # save updated data to dataset annual_pop:
  qri save --body /path/to/data.csv me/annual_pop

  # save updated dataset (no data) to annual_pop:
  qri save --file /path/to/dataset.yaml me/annual_pop
  
  # re-execute a dataset that has a transform:
  qri save me/tf_dataset`,
		Annotations: map[string]string{
			"group": "dataset",
		},
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

	cmd.Flags().StringSliceVarP(&o.FilePaths, "file", "f", nil, "dataset or component file (yaml or json)")
	cmd.Flags().StringVarP(&o.Title, "title", "t", "", "title of commit message for save")
	cmd.Flags().StringVarP(&o.Message, "message", "m", "", "commit message for save")
	cmd.Flags().StringVarP(&o.BodyPath, "body", "", "", "path to file or url of data to add as dataset contents")
	cmd.Flags().StringVarP(&o.Recall, "recall", "", "", "restore revisions from dataset history")
	// cmd.Flags().BoolVarP(&o.ShowValidation, "show-validation", "s", false, "display a list of validation errors upon adding")
	cmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	cmd.Flags().BoolVarP(&o.Publish, "publish", "p", false, "publish this dataset to the registry")
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "simulate saving a dataset")
	cmd.Flags().BoolVar(&o.Force, "force", false, "force a new commit, even if no changes are detected")
	cmd.Flags().BoolVarP(&o.KeepFormat, "keep-format", "k", false, "convert incoming data to stored data format")
	// TODO(dlong): --no-render is deprecated, viz are being phased out, in favor of readme.
	cmd.Flags().BoolVar(&o.NoRender, "no-render", false, "don't store a rendered version of the the vizualization ")
	cmd.Flags().BoolVarP(&o.NewName, "new", "n", false, "save a new dataset only, using an available name")

	return cmd
}

// SaveOptions encapsulates state for the save command
type SaveOptions struct {
	ioes.IOStreams

	Refs      *RefSelect
	FilePaths []string
	BodyPath  string
	Recall    string

	Title   string
	Message string

	UsingFSI       bool
	Replace        bool
	ShowValidation bool
	Publish        bool
	DryRun         bool
	KeepFormat     bool
	Force          bool
	NoRender       bool
	Secrets        []string
	NewName        bool

	DatasetRequests *lib.DatasetRequests
	FSIMethods      *lib.FSIMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *SaveOptions) Complete(f Factory, args []string) (err error) {
	if o.DatasetRequests, err = f.DatasetRequests(); err != nil {
		return
	}

	if o.Refs, err = GetCurrentRefSelect(f, args, 1, nil); err != nil {
		// Not an error to use an empty reference, it will be inferred later on.
		if err != repo.ErrEmptyRef {
			return err
		}
	}

	o.UsingFSI = o.Refs.IsLinked()

	// Make all paths absolute. Especially important if we are running
	// `qri connect` in a different terminal, and that instance is in a different directory;
	// that instance won't correctly find the body file we want to load if it's not absolute.
	for i := range o.FilePaths {
		if err = qfs.AbsPath(&o.FilePaths[i]); err != nil {
			return
		}
	}

	if err := qfs.AbsPath(&o.BodyPath); err != nil {
		return fmt.Errorf("body file: %s", err)
	}

	return nil
}

// Validate checks that all user input is valid
func (o *SaveOptions) Validate() error {
	return nil
}

// Run executes the save command
func (o *SaveOptions) Run() (err error) {
	printRefSelect(o.Out, o.Refs)

	o.StartSpinner()
	defer o.StopSpinner()

	// TODO (b5): cmd should never need to parse a dataset reference
	ref, err := parseCmdLineDatasetRef(o.Refs.Ref())
	if err != nil && len(o.FilePaths) == 0 {
		return lib.NewError(lib.ErrBadArgs, "error parsing dataset reference '"+o.Refs.Ref()+"'")
	}

	p := &lib.SaveParams{
		Ref:      ref.AliasString(),
		BodyPath: o.BodyPath,
		Title:    o.Title,
		Message:  o.Message,

		ReadFSI:             o.UsingFSI,
		WriteFSI:            o.UsingFSI,
		FilePaths:           o.FilePaths,
		Private:             false,
		Publish:             o.Publish,
		DryRun:              o.DryRun,
		Recall:              o.Recall,
		ConvertFormatToPrev: o.KeepFormat,
		Force:               o.Force,
		ReturnBody:          o.DryRun,
		ShouldRender:        !o.NoRender,
		NewName:             o.NewName,
	}

	if o.Secrets != nil {
		// Stop the spinner so the user can see the prompt, and the answer they type will
		// not be erased. Output the message to error stream in case stdout is captured.
		o.StopSpinner()
		if !confirm(o.ErrOut, o.In, `
Warning: You are providing secrets to a dataset transformation.
Never provide secrets to a transformation you do not trust.
continue?`, true) {
			return
		}
		// Restart the spinner.
		o.StartSpinner()
		if p.Secrets, err = parseSecrets(o.Secrets...); err != nil {
			return err
		}
	}

	res := &reporef.DatasetRef{}
	if err = o.DatasetRequests.Save(p, res); err != nil {
		return err
	}

	o.StopSpinner()
	printSuccess(o.ErrOut, "dataset saved: %s", res)
	if res.Dataset.Structure != nil && res.Dataset.Structure.ErrCount > 0 {
		printWarning(o.ErrOut, fmt.Sprintf("this dataset has %d validation errors", res.Dataset.Structure.ErrCount))
	}

	if o.DryRun {
		data, err := json.MarshalIndent(res.Dataset, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprint(o.Out, string(data))
	}

	return nil
}
