package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
	"io/ioutil"
)

// NewSaveCommand creates a `qri save` cobra command used for saving changes
// to datasets
func NewSaveCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &SaveOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "save",
		Aliases: []string{"update", "commit"},
		Short:   "save changes to a dataset",
		Long: `
Save is how you change a dataset, updating one or more of data, metadata, and 
structure. You can also update your data via url. Every time you run save, 
an entry is added to your dataset’s log (which you can see by running “qri log 
[ref]”). Every time you save, you can provide a message about what you changed 
and why. If you don’t provide a message 
qri will automatically generate one for you.

Currently you can only save changes to datasets that you control. Tools for 
collaboration are in the works. Sit tight sportsfans.`,
		Example: `  save updated data to dataset annual_pop:
  $ qri --body /path/to/data.csv me/annual_pop

  save updated dataset (no data) to annual_pop:
  $ qri --file /path/to/dataset.yaml me/annual_pop`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Run())
		},
	}

	cmd.Flags().StringVarP(&o.FilePath, "file", "f", "", "dataset data file (yaml or json)")
	cmd.Flags().StringVarP(&o.Title, "title", "t", "", "title of commit message for save")
	cmd.Flags().StringVarP(&o.Message, "message", "m", "", "commit message for save")
	cmd.Flags().StringVarP(&o.BodyPath, "body", "", "", "path to file or url of data to add as dataset contents")
	cmd.Flags().BoolVarP(&o.ShowValidation, "show-validation", "s", false, "display a list of validation errors upon adding")
	cmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	cmd.Flags().BoolVarP(&o.NoRegistry, "no-registry", "n", false, "don't publish this dataset to the registry")

	return cmd
}

// SaveOptions encapsulates state for the save command
type SaveOptions struct {
	IOStreams

	Ref            string
	FilePath       string
	BodyPath       string
	Title          string
	Message        string
	Passive        bool
	Rescursive     bool
	ShowValidation bool
	NoRegistry     bool
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

// Run executes the save command
func (o *SaveOptions) Run() (err error) {
	if o.Ref == "" && o.FilePath == "" {
		return fmt.Errorf("please provide the name of an existing dataset to save updates to, or specify a dataset --file with name and peername")
	}

	ref, err := parseCmdLineDatasetRef(o.Ref)
	if err != nil && o.FilePath == "" {
		return err
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

	if ref.Name != "" {
		dsp.Name = ref.Name
	}
	if ref.Peername != "" {
		dsp.Peername = ref.Peername
	} else if dsp.Peername == "" {
		dsp.Peername = "me"
	}

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
		Publish: !o.NoRegistry,
	}

	res := &repo.DatasetRef{}
	if err = o.DatasetRequests.Save(p, res); err != nil {
		return err
	}

	printSuccess(o.Out, "dataset saved: %s", res)
	if res.Dataset.Structure.ErrCount > 0 {
		printWarning(o.Out, fmt.Sprintf("this dataset has %d validation errors", res.Dataset.Structure.ErrCount))
	}
	return nil
}
