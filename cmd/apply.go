package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewApplyCommand creates an add command
func NewApplyCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ApplyOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "apply DATASET",
		Aliases: []string{"add"},
		Short:   "",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f); err != nil {
				return err
			}
			return o.Run(args)
		},
	}

	cmd.Flags().StringVar(&o.ScriptPath, "script", "", "path to directory to transform script file")
	cmd.MarkFlagRequired("script")
	cmd.Flags().StringSliceVar(&o.Config, "config", nil, "key,value,key,value sequence of configuration")
	cmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	// cmd.Flags().StringVar(&o.Remote, "remote", "", "location to pull from")
	// cmd.MarkFlagFilename("link")
	// cmd.Flags().BoolVar(&o.LogsOnly, "logs-only", false, "only fetch logs, skipping HEAD data")

	return cmd
}

// ApplyOptions encapsulates state for the add command
type ApplyOptions struct {
	ioes.IOStreams
	ScriptPath string
	Config     []string
	Secrets    []string

	// Remote         string
	TransformMethods *lib.TransformMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ApplyOptions) Complete(f Factory) (err error) {
	if o.TransformMethods, err = f.TransformMethods(); err != nil {
		return err
	}

	o.ScriptPath, err = filepath.Abs(o.ScriptPath)
	if err != nil {
		return err
	}
	return nil
}

// Run adds another peer's dataset to this user's repo
func (o *ApplyOptions) Run(args []string) (err error) {
	var refstr string
	if len(args) > 0 {
		refstr = args[0]
	}

	tf := &dataset.Transform{
		ScriptPath: o.ScriptPath,
	}

	if len(o.Config) > 0 {
		// tf.Config, err = parseSecrets(o.Config...)
		// if err != nil {
		// 	return err
		// }
	}

	if len(o.Secrets) > 0 {
		tf.Secrets, err = parseSecrets(o.Secrets...)
		if err != nil {
			return err
		}
	}

	p := &lib.ApplyParams{
		RefString: refstr,
		Transform: tf,
	}

	res := &dataset.Dataset{}
	if err := o.TransformMethods.Apply(p, res); err != nil {
		return err
	}

	data, err := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(data))
	return err
}
