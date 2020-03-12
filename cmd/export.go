package cmd

import (
	"fmt"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewExportCommand creates a new export cobra command
// exportCmd represents the export command
func NewExportCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ExportOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "export DATASET",
		Short: "Copy datasets to your local filesystem",
		Long: `
Export gets datasets out of qri. By default it exports the dataset body, as ` + "`body.csv`" + `, header as` + "`dataset.json`" + `, and ref, as ` + "`ref.txt`" + ` files. 

To export to a specific directory, use the --output flag.`,
		Example: `  # export dataset
  qri export me/annual_pop

  # export to a specific directory
  qri export -o ~/new_directory me/annual_pop`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "path to write to, default is current directory")
	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "format for the exported dataset, such as native, json, xlsx. default: json")
	cmd.Flags().BoolVarP(&o.Zipped, "zip", "z", false, "export as a zip file")

	return cmd
}

// ExportOptions encapsulates state for the export command
type ExportOptions struct {
	ioes.IOStreams

	Refs   *RefSelect
	Output string
	Format string
	Zipped bool

	UsingRPC       bool
	ExportRequests *lib.ExportRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ExportOptions) Complete(f Factory, args []string) (err error) {
	if o.Refs, err = GetCurrentRefSelect(f, args, 1, nil); err != nil {
		if err != repo.ErrEmptyRef {
			return err
		}
	}
	if f.RPC() != nil {
		return usingRPCError("export")
	}
	o.ExportRequests, err = f.ExportRequests()
	return err
}

// Run executes the Export command
func (o *ExportOptions) Run() error {
	path := o.Output
	format := o.Format

	p := &lib.ExportParams{
		Ref:    o.Refs.Ref(),
		Output: path,
		Format: format,
		Zipped: o.Zipped,
	}

	var fileWritten string
	if err := o.ExportRequests.Export(p, &fileWritten); err != nil {
		return err
	}

	fmt.Fprintf(o.Out, "dataset exported to \"%s\"\n", fileWritten)

	return nil
}
