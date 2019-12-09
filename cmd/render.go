package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewRenderCommand creates a new `qri render` command for executing templates against datasets
func NewRenderCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &RenderOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render a dataset readme or a dataset template",
		Long: `
Render a dataset either by converting its readme from markdown to
html, or by filling in a template using the go/html template style.

Use the ` + "`--output`" + ` flag to save the rendered html to a file.

Use the ` + "`--viz`" + ` flag to render the viz. Default is to use readme.

Use the ` + "`--template`" + ` flag to use a custom template. If no template is
provided, Qri will render the dataset with a default template.`,
		Example: `  render the readme of a dataset called me/schools:
  $ qri render -o=schools.html me/schools

  render a dataset with a custom template:
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
	cmd.Flags().BoolVarP(&o.UseViz, "viz", "v", false, "whether to use the viz component")
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "path to write output file")

	return cmd
}

// RenderOptions encapsulates state for the render command
type RenderOptions struct {
	ioes.IOStreams

	Refs     *RefSelect
	Template string
	UseViz   bool
	Output   string

	RenderRequests *lib.RenderRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RenderOptions) Complete(f Factory, args []string) (err error) {
	if o.RenderRequests, err = f.RenderRequests(); err != nil {
		return err
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, 1, nil); err != nil {
		return err
	}
	return nil
}

// Run executes the render command
func (o *RenderOptions) Run() error {
	if o.Template != "" && !o.UseViz {
		return fmt.Errorf("can not specify both --template without --viz flag")
	}

	if o.UseViz {
		return o.RunVizRender()
	}

	return o.RunReadmeRender()
}

// RunVizRender renders a viz component of a dataset as html
func (o *RenderOptions) RunVizRender() (err error) {
	var template []byte
	if o.Template != "" {
		template, err = ioutil.ReadFile(o.Template)
		if err != nil {
			return err
		}
	}

	p := &lib.RenderParams{
		Ref:       o.Refs.Ref(),
		Template:  template,
		OutFormat: "html",
	}

	res := []byte{}
	if err := o.RenderRequests.RenderViz(p, &res); err != nil {
		if err == repo.ErrEmptyRef {
			return lib.NewError(err, "peername and dataset name needed in order to render, for example:\n   $ qri render me/dataset_name\nsee `qri render --help` from more info")
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

// RunReadmeRender renders a readme file as html
func (o *RenderOptions) RunReadmeRender() error {
	printRefSelect(o.Out, o.Refs)

	p := &lib.RenderParams{
		Ref:       o.Refs.Ref(),
		UseFSI:    o.Refs.IsLinked(),
		OutFormat: "html",
	}

	var res string
	if err := o.RenderRequests.RenderReadme(p, &res); err != nil {
		return err
	}

	if o.Output == "" {
		fmt.Fprint(o.Out, res)
	} else {
		ioutil.WriteFile(o.Output, []byte(res), 0777)
	}
	return nil
}
