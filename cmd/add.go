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
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewAddCommand creates a new add command
func NewAddCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &AddOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:        "add",
		Short:      "Add a dataset",
		SuggestFor: []string{"init"},
		Annotations: map[string]string{
			"group": "dataset",
		},
		Long: `
Add creates a new dataset from data you supply. Please note that all data added 
to qri is made public on the distributed web when you run qri connect.

When adding data, you can supply metadata and dataset structure, but it’s not 
required. qri does what it can to infer the details you don’t provide. 
add currently supports three data formats:
- CSV  (Comma Separated Values)
- JSON (Javascript Object Notation)
- CBOR (Concise Binary Object Representation)

Once you’ve added data, you can use the export command to pull the data out of 
qri, change the data outside of qri, and use the save command to record those 
changes to qri.`,
		Example: `  add a new dataset named annual_pop:
  $ qri add --data data.csv me/annual_pop

  create a dataset with a dataset data file:
  $ qri add --file dataset.yaml --data comics.csv me/comic_characters`,
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f))
			ExitIfErr(o.Run(args))
		},
	}

	cmd.Flags().StringVarP(&o.File, "file", "f", "", "dataset data file in either yaml or json format")
	cmd.Flags().StringVarP(&o.DataPath, "data", "d", "", "path to file or url to initialize from")
	cmd.Flags().StringVarP(&o.Title, "title", "t", "", "commit title")
	cmd.Flags().StringVarP(&o.Message, "message", "m", "", "commit message")
	cmd.Flags().BoolVarP(&o.Private, "private", "", false, "make dataset private. WARNING: not yet implimented. Please refer to https://github.com/qri-io/qri/issues/291 for updates")
	cmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	cmd.Flags().BoolVarP(&o.Publish, "publish", "p", false, "publish this dataset to the registry")
	// datasetAddCmd.Flags().BoolVarP(&o.ShowValidation, "show-validation", "s", false, "display a list of validation errors upon adding")

	return cmd
}

// AddOptions encapsulates state for the add command
type AddOptions struct {
	IOStreams

	File           string
	DataPath       string
	Name           string
	Title          string
	Message        string
	Passive        bool
	ShowValidation bool
	Private        bool
	Publish        bool
	Secrets        []string

	DatasetRequests *core.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *AddOptions) Complete(f Factory) (err error) {
	if o.DatasetRequests, err = f.DatasetRequests(); err != nil {
		return
	}
	return nil
}

// Run adds a dataset, parsing references
func (o *AddOptions) Run(args []string) error {
	ingest := (o.File != "" || o.DataPath != "")

	if ingest {
		var arg string
		if len(args) == 1 {
			arg = args[0]
		}
		ref, _ := repo.ParseDatasetRef(arg)
		return o.InitDataset(ref)
	}

	for _, arg := range args {
		ref, err := repo.ParseDatasetRef(arg)
		if err != nil {
			return err
		}

		res := repo.DatasetRef{}
		if err = o.DatasetRequests.Add(&ref, &res); err != nil {
			return err
		}

		printDatasetRefInfo(o.Out, 1, res)
		printInfo(o.Out, "Successfully added dataset %s", ref)
	}

	return nil
}

// InitDataset creates a dataset from input data
func (o *AddOptions) InitDataset(name repo.DatasetRef) error {
	var err error

	dsp := &dataset.DatasetPod{}
	if o.File != "" {
		f, err := os.Open(o.File)
		if err != nil {
			return err
		}

		switch strings.ToLower(filepath.Ext(o.File)) {
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

	if name.Peername != "" {
		dsp.Name = name.Name
	}
	if name.Peername != "" {
		dsp.Peername = name.Peername
	}
	if o.DataPath != "" {
		if o.DataPath, err = filepath.Abs(o.DataPath); err != nil {
			return err
		}
		dsp.DataPath = o.DataPath
	}
	if dsp.Transform != nil {
		if o.Secrets != nil {
			if !confirm(o.Out, o.In, `
Warning: You are providing secrets to a dataset transformation.
Never provide secrets to a transformation you do not trust.
continue?`, true) {
				return nil
			}
			if dsp.Transform.Secrets, err = parseSecrets(o.Secrets...); err != nil {
				return err
			}
		}
		if dsp.Transform.ScriptPath != "" {
			if dsp.Transform.ScriptPath, err = filepath.Abs(dsp.Transform.ScriptPath); err != nil {
				return err
			}
		}
	}

	if dsp.Commit == nil && (o.Title != "" || o.Message != "") {
		dsp.Commit = &dataset.CommitPod{}
	}

	if o.Title != "" {
		dsp.Commit.Title = o.Title
	}

	if o.Message != "" {
		dsp.Commit.Message = o.Message
	}

	p := &core.SaveParams{
		Dataset: dsp,
		Private: o.Private,
		Publish: o.Publish,
	}

	ref := repo.DatasetRef{}
	if err = o.DatasetRequests.Init(p, &ref); err != nil {
		return err
	}

	if ref.Dataset.Structure.ErrCount > 0 {
		printWarning(o.Out, fmt.Sprintf("this dataset has %d validation errors", ref.Dataset.Structure.ErrCount))

		// TODO - restore.
		// if o.ShowValidation {
		// 	printWarning(o.Out, "Validation Error Detail:")
		// 	data, err := ioutil.ReadAll(dataFile)
		// 	ExitIfErr(err)
		// 	ds, err := ref.DecodeDataset()
		// 	ErrExit(err)
		// 	errorList, err := ds.Structure.Schema.ValidateBytes(data)
		// 	ExitIfErr(err)
		// 	for i, validationErr := range errorList {
		// 		printWarning(o.Out, fmt.Sprintf("\t%d. %s", i+1, validationErr.Error()))
		// 	}
		// }
	}

	ref.Peername = "me"
	printSuccess(o.Out, "added new dataset %s", ref)
	return nil
}
