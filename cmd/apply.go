package cmd

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewApplyCommand creates a new `qri apply` cobra command for applying transformations
func NewApplyCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ApplyOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "apply a transform to a dataset",
		Long: `Apply runs a transform script. The result of the transform is displayed after
the command completes.

Nothing is saved in the user's repository.`,
		Example: `  # Apply a transform and display the output:
 $ qri apply --script transform.star`,
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

	cmd.Flags().StringVar(&o.FilePath, "file", "", "path of transform script file")
	cmd.MarkFlagRequired("file")
	cmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")

	return cmd
}

// ApplyOptions encapsulates state for the apply command
type ApplyOptions struct {
	ioes.IOStreams

	Refs     *RefSelect
	FilePath string
	Secrets  []string

	TransformMethods *lib.TransformMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ApplyOptions) Complete(f Factory, args []string) (err error) {
	if o.TransformMethods, err = f.TransformMethods(); err != nil {
		return err
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, -1, nil); err != nil {
		// This error will be handled during validation
		if err != repo.ErrEmptyRef {
			return err
		}
		err = nil
	}
	o.FilePath, err = filepath.Abs(o.FilePath)
	if err != nil {
		return err
	}
	return nil
}

// Run executes the apply command
func (o *ApplyOptions) Run() error {
	printRefSelect(o.ErrOut, o.Refs)

	var err error

	if !strings.HasSuffix(o.FilePath, ".star") {
		return errors.New("only transform scripts are supported by --file")
	}

	tf := dataset.Transform{
		ScriptPath: o.FilePath,
	}

	if len(o.Secrets) > 0 {
		tf.Secrets, err = parseSecrets(o.Secrets...)
		if err != nil {
			return err
		}
	}

	params := lib.ApplyParams{
		Refstr:       o.Refs.Ref(),
		Transform:    &tf,
		ScriptOutput: o.Out,
	}
	res := lib.ApplyResult{}
	if err = o.TransformMethods.Apply(&params, &res); err != nil {
		return err
	}

	data, err := json.MarshalIndent(res.Data, "", " ")
	if err != nil {
		return err
	}
	printSuccess(o.Out, string(data))
	return nil
}
