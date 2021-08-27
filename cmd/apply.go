package cmd

import (
	"context"
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

The apply command itself does not commit results to the repository. Use
the --apply flag on the save command to commit results from transforms.`,
		Example: ` # Apply a transform and display the output:
 $ qri apply --file transform.star

 # Apply a transform using an existing dataset version:
 $ qri apply --file transform.star me/my_dataset`,
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

	Instance *lib.Instance

	Refs     *RefSelect
	FilePath string
	Secrets  []string
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ApplyOptions) Complete(f Factory, args []string) (err error) {
	if o.Instance, err = f.Instance(); err != nil {
		return err
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, -1); err != nil {
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
func (o *ApplyOptions) Run() (err error) {

	if !strings.HasSuffix(o.FilePath, ".star") {
		return errors.New("only transform scripts are supported by --file")
	}

	ctx := context.TODO()
	inst := o.Instance

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
		Ref:          o.Refs.Ref(),
		Transform:    &tf,
		ScriptOutput: o.Out,
		Wait:         true,
	}

	res, err := inst.Automation().Apply(ctx, &params)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(res.Data, "", " ")
	if err != nil {
		return err
	}
	printSuccess(o.Out, string(data))
	return nil
}
