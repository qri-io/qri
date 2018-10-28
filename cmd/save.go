package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
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
will re execute the transform. To only re-run the tranform, run save with no args.

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

	cmd.Flags().StringVarP(&o.FilePath, "file", "f", "", "dataset data file (yaml or json)")
	cmd.Flags().StringVarP(&o.Title, "title", "t", "", "title of commit message for save")
	cmd.Flags().StringVarP(&o.Message, "message", "m", "", "commit message for save")
	cmd.Flags().StringVarP(&o.BodyPath, "body", "", "", "path to file or url of data to add as dataset contents")
	// cmd.Flags().BoolVarP(&o.ShowValidation, "show-validation", "s", false, "display a list of validation errors upon adding")
	cmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	cmd.Flags().BoolVarP(&o.Publish, "publish", "p", false, "publish this dataset to the registry")
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "simulate saving a dataset")

	return cmd
}

// SaveOptions encapsulates state for the save command
type SaveOptions struct {
	ioes.IOStreams

	Ref            string
	FilePath       string
	BodyPath       string
	Title          string
	Message        string
	Passive        bool
	Rescursive     bool
	ShowValidation bool
	Publish        bool
	DryRun         bool
	Secrets        []string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *SaveOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		o.Ref = args[0]
	}

	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Validate checks that all user input is valid
func (o *SaveOptions) Validate() error {
	return nil
}

// Run executes the save command
func (o *SaveOptions) Run() (err error) {
	o.StartSpinner()
	defer o.StopSpinner()

	ref, err := parseCmdLineDatasetRef(o.Ref)
	if err != nil && o.FilePath == "" {
		return lib.NewError(lib.ErrBadArgs, "error parsing dataset reference '"+o.Ref+"'")
	}

	dsp := &dataset.DatasetPod{}
	if o.FilePath != "" {
		f, err := os.Open(o.FilePath)
		if err != nil {
			return err
		}

		switch strings.ToLower(filepath.Ext(o.FilePath)) {
		case ".yaml", ".yml":
			data, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			if err = dsutil.UnmarshalYAMLDatasetPod(data, dsp); err != nil {
				return err
			}
		case ".json":
			if err = json.NewDecoder(f).Decode(dsp); err != nil {
				return err
			}
		}
	}

	dsp.Name = ref.Name
	dsp.Peername = ref.Peername
	if (o.Title != "" || o.Message != "") && dsp.Commit == nil {
		dsp.Commit = &dataset.CommitPod{}
	}
	if o.Title != "" {
		dsp.Commit.Title = o.Title
	}
	if o.Message != "" {
		dsp.Commit.Message = o.Message
	}

	if o.BodyPath != "" {
		dsp.BodyPath = o.BodyPath
	}
	if dsp.BodyPath != "" {
		if _, err := os.Stat(dsp.BodyPath); os.IsNotExist(err) {
			return fmt.Errorf("body file \"%s\": no such file or directory", dsp.BodyPath)
		}
		// Get the absolute path to the body file. Especially important if we are running
		// `qri connect` in a different terminal, and that instance is in a different directory;
		// that instance won't correctly find the body file we want to load if it's not absolute.
		dsp.BodyPath, err = filepath.Abs(dsp.BodyPath)
		if err != nil {
			return err
		}
	}
	if dsp.Transform != nil && o.Secrets != nil {
		if !confirm(o.Out, o.In, `
Warning: You are providing secrets to a dataset transformation.
Never provide secrets to a transformation you do not trust.
continue?`, true) {
			return
		}

		if dsp.Transform.Secrets, err = parseSecrets(o.Secrets...); err != nil {
			return err
		}
	}

	p := &lib.SaveParams{
		Dataset: dsp,
		Private: false,
		Publish: o.Publish,
		DryRun:  o.DryRun,
	}

	res := &repo.DatasetRef{}
	if err = o.DatasetRequests.Save(p, res); err != nil {
		return err
	}

	o.StopSpinner()
	printSuccess(o.Out, "dataset saved: %s", res)
	if res.Dataset.Structure.ErrCount > 0 {
		printWarning(o.Out, fmt.Sprintf("this dataset has %d validation errors", res.Dataset.Structure.ErrCount))
	}
	return nil
}
