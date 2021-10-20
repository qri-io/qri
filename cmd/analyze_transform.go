package cmd

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewAnalyzeTransformCommand creates a new `qri analyze-transform` cobra command
func NewAnalyzeTransformCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &AnalyzeTransformOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "analyze_transform",
		Short: "analyze a transform script",
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

	return cmd
}

// AnalyzeTransformOptions encapsulates state for the analyze-transform command
type AnalyzeTransformOptions struct {
	ioes.IOStreams
	Instance *lib.Instance
	FilePath string
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *AnalyzeTransformOptions) Complete(f Factory, args []string) (err error) {
	if o.Instance, err = f.Instance(); err != nil {
		return err
	}
	o.FilePath, err = filepath.Abs(o.FilePath)
	return err
}

// Run executes the analyze-transform command
func (o *AnalyzeTransformOptions) Run() (err error) {
	if !strings.HasSuffix(o.FilePath, ".star") {
		return errors.New("only transform scripts are supported by --file")
	}

	ctx := context.TODO()
	inst := o.Instance

	params := lib.AnalyzeTransformParams{
		ScriptFileName: o.FilePath,
	}

	res, err := inst.Automation().AnalyzeTransform(ctx, &params)
	if err != nil {
		return err
	}

	for _, msg := range res.Diagnostics {
		if msg.Category == "unused" {
			printWarning(o.Out, "Function unused: %s", msg.Message)
		} else {
			printWarning(o.Out, "Unknown warning: %s", msg.Message)
		}
	}
	return nil
}
