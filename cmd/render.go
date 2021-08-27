package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/dsref"
	qerr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewRenderCommand creates a new `qri render` command for executing templates against datasets
func NewRenderCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &RenderOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "render",
		Short: "render a dataset readme or a dataset template",
		Long: `Render a dataset either by converting its readme from markdown to
html, or by filling in a template using the go/html template style.

Use the ` + "`--output`" + ` flag to save the rendered html to a file.

Use the ` + "`--viz`" + ` flag to render the viz. Default is to use readme.

Use the ` + "`--template`" + ` flag to use a custom template. If no template is
provided, Qri will render the dataset with a default template.`,
		Example: `  # Render the readme of a dataset called me/schools:
  $ qri render -o=schools.html me/schools

  # Render a dataset with a custom template:
  $ qri render --viz --template=template.html me/schools`,
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

	cmd.Flags().StringVarP(&o.Template, "template", "t", "", "path to template file")
	cmd.MarkFlagFilename("template")
	cmd.Flags().BoolVarP(&o.UseViz, "viz", "v", false, "whether to use the viz component")
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "path to write output file")
	cmd.MarkFlagFilename("output")

	return cmd
}

// RenderOptions encapsulates state for the render command
type RenderOptions struct {
	ioes.IOStreams

	Refs     *RefSelect
	Template string
	UseViz   bool
	Output   string

	inst *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RenderOptions) Complete(f Factory, args []string) (err error) {
	if o.inst, err = f.Instance(); err != nil {
		return err
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, 1); err != nil {
		return err
	}
	return nil
}

// Run executes the render command
func (o *RenderOptions) Run() error {
	// NOTE: `--viz` is required even if we could infer it from `--template` in
	// order to make extra sure the user is not mixing up possible args when
	// rendering the readme.
	if o.Template != "" && !o.UseViz {
		return fmt.Errorf("you must specify --viz when using --template")
	}

	p := &lib.RenderParams{}
	var err error
	if o.UseViz {
		p, err = o.vizRenderParams()
		if err != nil {
			return err
		}
	} else {
		p = o.readmeRenderParams()
	}

	res, err := o.inst.Dataset().Render(context.TODO(), p)
	if err != nil {
		if errors.Is(err, dsref.ErrEmptyRef) {
			return qerr.New(err, "peername and dataset name needed in order to render, for example:\n   $ qri render me/dataset_name\nsee `qri render --help` from more info")
		}
		return err
	}

	if o.Output == "" {
		fmt.Fprint(o.Out, string(res))
	} else {
		ioutil.WriteFile(o.Output, res, 0777)
	}
	return nil
}

func (o *RenderOptions) vizRenderParams() (p *lib.RenderParams, err error) {
	var template []byte
	if o.Template != "" {
		template, err = ioutil.ReadFile(o.Template)
		if err != nil {
			return nil, err
		}
	}

	return &lib.RenderParams{
		Ref:      o.Refs.Ref(),
		Template: template,
		Format:   "html",
		Selector: "viz",
	}, nil
}

func (o *RenderOptions) readmeRenderParams() *lib.RenderParams {
	return &lib.RenderParams{
		Ref:      o.Refs.Ref(),
		Format:   "html",
		Selector: "readme",
	}
}
