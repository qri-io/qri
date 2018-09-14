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
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// TODO: Tests.

const providingSecretWarningMessage = `
Warning: You are providing secrets to a dataset transformation.
Never provide secrets to a transformation you do not trust.
continue?`

// NewNewCommand creates a new command
func NewNewCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &NewOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:        "new",
		Short:      "Create a new dataset",
		SuggestFor: []string{"init"},
		Annotations: map[string]string{
			"group": "dataset",
		},
		Long: `
New creates a dataset from data you supply. Please note that all data added
to qri is made public on the distributed web when you run qri connect

When adding data, you can supply metadata and dataset structure, but it’s not
required. qri does what it can to infer the details you don’t provide.
qri currently supports three data formats:
- CSV  (Comma Separated Values)
- JSON (Javascript Object Notation)
- CBOR (Concise Binary Object Representation)

Once you’ve added data, you can use the export command to pull the data out of
qri, change the data outside of qri, and use the save command to record those
changes to qri.`,
		Example: `  create a new dataset named annual_pop:
  $ qri new --body data.csv me/annual_pop

create a dataset with a dataset data file:
  $ qri new --file dataset.yaml --body comics.csv me/comic_characters`,
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.ErrOut, o.Complete(f))
			ExitIfErr(o.ErrOut, o.Run(args))
		},
	}

	cmd.Flags().StringVarP(&o.File, "file", "f", "", "dataset data file in either yaml or json format")
	cmd.Flags().StringVarP(&o.BodyPath, "body", "b", "", "path to file or url for contents of dataset")
	cmd.Flags().StringVarP(&o.Title, "title", "t", "", "commit title")
	cmd.Flags().StringVarP(&o.Message, "message", "m", "", "commit message")
	cmd.Flags().BoolVarP(&o.Private, "private", "", false, "make dataset private. WARNING: not yet implimented. Please refer to https://github.com/qri-io/qri/issues/291 for updates")
	cmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	cmd.Flags().BoolVarP(&o.Publish, "publish", "p", false, "publish this dataset to the registry")

	return cmd
}

// NewOptions encapsulates state for the new command
type NewOptions struct {
	IOStreams

	File     string
	BodyPath string
	Title    string
	Message  string
	Private  bool
	Publish  bool
	Secrets  []string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *NewOptions) Complete(f Factory) (err error) {
	if o.DatasetRequests, err = f.DatasetRequests(); err != nil {
		return
	}
	return nil
}

// Run creates a new dataset
func (o *NewOptions) Run(args []string) (err error) {
	spinner.Start()
	defer spinner.Stop()

	if o.File == "" && o.BodyPath == "" {
		return fmt.Errorf("creating new dataset needs either --file or --body")
	}

	var arg string
	if len(args) == 1 {
		arg = args[0]
	}
	ref, _ := parseCmdLineDatasetRef(arg)

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

	if ref.Name != "" {
		dsp.Name = ref.Name
	}
	if ref.Peername != "" {
		dsp.Peername = ref.Peername
	}
	if o.BodyPath != "" {
		if o.BodyPath, err = filepath.Abs(o.BodyPath); err != nil {
			return err
		}
		dsp.BodyPath = o.BodyPath
	}
	if dsp.Transform != nil {
		if o.Secrets != nil {
			if !confirm(o.Out, o.In, providingSecretWarningMessage, true) {
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

	if dsp.Viz != nil {
		if dsp.Viz.ScriptPath != "" {
			if dsp.Viz.ScriptPath, err = filepath.Abs(dsp.Viz.ScriptPath); err != nil {
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

	p := &lib.SaveParams{
		Dataset: dsp,
		Private: o.Private,
		Publish: o.Publish,
	}

	ref = repo.DatasetRef{}
	if err = o.DatasetRequests.New(p, &ref); err != nil {
		return err
	}

	if ref.Dataset.Structure.ErrCount > 0 {
		printWarning(o.Out, fmt.Sprintf("this dataset has %d validation errors", ref.Dataset.Structure.ErrCount))
	}

	ref.Peername = "me"
	printSuccess(o.Out, "created new dataset %s", ref)
	return nil
}
