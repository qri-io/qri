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

Use the ` + "`--template`" + ` flag to use a custom template. If no template is
provided, Qri will render the dataset with a default template.

Use the ` + "`--readme`" + ` flag to convert the dataset's readme
component from markdown to html.`,
		Example: `  render a dataset called me/schools:
  $ qri render -o=schools.html me/schools

  render a dataset with a custom template:
  $ qri render --template=template.html me/schools

  render a dataset's readme file:
  $ qri render --readme me/schools`,
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
	cmd.Flags().BoolVarP(&o.UseReadme, "readme", "r", false, "whether to use the readme component")
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "path to write output file")

	return cmd
}

// RenderOptions encapsulates state for the render command
type RenderOptions struct {
	ioes.IOStreams

	Refs      *RefSelect
	Template  string
	UseReadme bool
	Output    string

	RenderRequests *lib.RenderRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RenderOptions) Complete(f Factory, args []string) (err error) {
	if o.RenderRequests, err = f.RenderRequests(); err != nil {
		return err
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, 1); err != nil {
		return err
	}
	return
}

// Run executes the render command
func (o *RenderOptions) Run() error {
	if o.Template != "" && o.UseReadme {
		return fmt.Errorf("can not specify both --template and --readme flags")
	}

	if o.UseReadme {
		return o.RunReadmeRender()
	}

	return o.RunTemplateRender()
}

// RunTemplateRender renders a dataset template as html
func (o *RenderOptions) RunTemplateRender() (err error) {
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
	if err := o.RenderRequests.RenderTemplate(p, &res); err != nil {
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
