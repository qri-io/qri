package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewSaveCommand creates a `qri save` cobra command used for saving changes
// to datasets
func NewSaveCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &SaveOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "save [DATASET]",
		Aliases: []string{"commit"},
		Short:   "save changes to a dataset",
		Long: `
Save is how you change a dataset, updating one or more dataset components. Every
time you run save, an entry is added to your dataset’s log (which you can see by
running `[1:] + "`qri log <dataset_reference>`" + `).

Dataset changes can be automated with a transform component adn the --apply flag
For more on transforms see https://qri.io/docs/transforms/overview
If the dataset you're changing has a transform, running ` + "`qri save --apply`" +
			`
will re-execute it to produce a new version

Every time you save, you can provide a message about what you changed and why. 
If you don’t provide a message Qri will automatically generate one for you.
The ` + "`--message`" + `" and ` + "`--title`" + ` flags allow you to add a 
commit message and title to the save.

When you make an update and save a dataset that you originally added from a 
different peer, the dataset gets renamed from ` + "`peers_name/dataset_name`" +
			` to
` + "`my_name/dataset_name`" + `.`,
		Example: `  # Save updated data to dataset annual_pop:
  $ qri save --body /path/to/data.csv me/annual_pop

  # Save updated dataset (no data) to annual_pop:
  $ qri save --file /path/to/dataset.yaml me/annual_pop
  
  # Re-execute the latest transform from history:
  $ qri save --apply me/tf_dataset`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.MaximumNArgs(1),
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
	cmd.MarkFlagFilename("file", "yaml", "yml", "json")
	cmd.Flags().StringVarP(&o.Title, "title", "t", "", "title of commit message for save")
	cmd.Flags().StringVarP(&o.Message, "message", "m", "", "commit message for save")
	cmd.Flags().StringVarP(&o.BodyPath, "body", "", "", "path to file or url of data to add as dataset contents")
	cmd.MarkFlagFilename("body")
	// cmd.Flags().BoolVarP(&o.ShowValidation, "show-validation", "s", false, "display a list of validation errors upon adding")
	cmd.Flags().BoolVar(&o.Apply, "apply", false, "apply a transformation and save the result")
	cmd.Flags().BoolVar(&o.NoApply, "no-apply", false, "don't apply any transforms that are added")
	cmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	cmd.Flags().BoolVar(&o.DeprecatedDryRun, "dry-run", false, "deprecated: use `qri apply` instead")
	cmd.Flags().BoolVar(&o.Force, "force", false, "force a new commit, even if no changes are detected")
	cmd.Flags().BoolVarP(&o.KeepFormat, "keep-format", "k", false, "convert incoming data to stored data format")
	// TODO(dustmop): --no-render is deprecated, viz are being phased out, in favor of readme.
	cmd.Flags().BoolVar(&o.NoRender, "no-render", false, "don't store a rendered version of the the visualization")
	cmd.Flags().BoolVarP(&o.NewName, "new", "n", false, "save a new dataset only, using an available name")
	cmd.Flags().StringVar(&o.Drop, "drop", "", "comma-separated list of components to remove")

	return cmd
}

// SaveOptions encapsulates state for the save command
type SaveOptions struct {
	ioes.IOStreams

	Refs      *RefSelect
	FilePaths []string
	BodyPath  string
	Drop      string

	Title   string
	Message string

	Apply            bool
	NoApply          bool
	DeprecatedDryRun bool
	Secrets          []string

	Replace        bool
	ShowValidation bool
	KeepFormat     bool
	Force          bool
	NoRender       bool
	NewName        bool
	UseDscache     bool

	inst *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *SaveOptions) Complete(f Factory, args []string) (err error) {
	if o.DeprecatedDryRun {
		return fmt.Errorf("--dry-run has been removed, use `qri apply` command instead")
	}

	if o.inst, err = f.Instance(); err != nil {
		return
	}

	if o.Refs, err = GetCurrentRefSelect(f, args, BadUpperCaseOkayWhenSavingExistingDataset, nil); err != nil {
		// Not an error to use an empty reference, it will be inferred later on.
		if err != repo.ErrEmptyRef {
			return
		}
	}

	return nil
}

// Validate checks that all user input is valid
func (o *SaveOptions) Validate() error {
	return nil
}

// Run executes the save command
func (o *SaveOptions) Run() (err error) {
	printRefSelect(o.ErrOut, o.Refs)

	p := &lib.SaveParams{
		Ref:      o.Refs.Ref(),
		BodyPath: o.BodyPath,
		Title:    o.Title,
		Message:  o.Message,

		ScriptOutput: o.ErrOut,
		FilePaths:    o.FilePaths,
		Private:      false,
		Apply:        o.Apply,
		Drop:         o.Drop,

		ConvertFormatToPrev: o.KeepFormat,
		Force:               o.Force,

		ShouldRender: !o.NoRender,
		NewName:      o.NewName,
	}

	// Check if file ends in '.star'. If so, either Apply or NoApply is required.
	// Apply is passed down to the lib level, NoApply ends here. NoApply's only purpose
	// is to ensure that the user wants to add a transform without running it, and explicitly
	// agrees that they're not expecting the old behavior: wherein adding a transform
	// would always run it.
	for _, file := range o.FilePaths {
		if strings.HasSuffix(file, ".star") {
			if !o.Apply && !o.NoApply {
				return fmt.Errorf("saving with a new transform requires either --apply or --no-apply flag")
			}
		}
	}

	// TODO(dustmop): Support component files, like .json and .yaml, which can contain
	// transform scripts.

	if o.Secrets != nil {
		if !confirm(o.ErrOut, o.In, `
Warning: You are providing secrets to a dataset transformation.
Never provide secrets to a transformation you do not trust.
continue?`, true) {
			return
		}
		if p.Secrets, err = parseSecrets(o.Secrets...); err != nil {
			return err
		}
	}

	ctx := context.TODO()
	res, err := o.inst.Dataset().Save(ctx, p)
	if err != nil {
		return err
	}

	ref := dsref.ConvertDatasetToVersionInfo(res).SimpleRef()
	ref.ProfileID = ""
	printSuccess(o.ErrOut, "dataset saved: %s", ref.String())
	if res.Structure != nil && res.Structure.ErrCount > 0 {
		printWarning(o.ErrOut, fmt.Sprintf("this dataset has %d validation errors", res.Structure.ErrCount))
	}

	return nil
}
