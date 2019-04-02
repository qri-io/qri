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
		Short: "Execute a template against a dataset",
		Long: `
You can use html templates, formatted in the go/html template style, 
to render visualizations from your dataset. These visualizations can be charts, 
graphs, or just display your dataset in a different format.

Use the ` + "`--output`" + ` flag to save the rendered html to a file.

Use the ` + "`--template`" + ` flag to use a custom template. If no template is
provided, Qri will render the dataset with a default template.`,
		Example: `  render a dataset called me/schools:
  $ qri render -o=schools.html me/schools

  render a dataset with a custom template:
  $ qri render --template=template.html me/schools`,
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
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "path to write output file")

	return cmd
}

// RenderOptions encapsulates state for the render command
type RenderOptions struct {
	ioes.IOStreams

	Ref      string
	Template string
	Output   string

	RenderRequests *lib.RenderRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *RenderOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		o.Ref = args[0]
	}
	o.RenderRequests, err = f.RenderRequests()
	return
}

// Run executes the render command
func (o *RenderOptions) Run() (err error) {
	var template []byte

	if o.Template != "" {
		template, err = ioutil.ReadFile(o.Template)
		if err != nil {
			return err
		}
	}

	p := &lib.RenderParams{
		Ref:            o.Ref,
		Template:       template,
		TemplateFormat: "html",
	}

	res := []byte{}
	if err = o.RenderRequests.Render(p, &res); err != nil {
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
