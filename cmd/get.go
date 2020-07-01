package cmd

import (
	"bytes"
	"fmt"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewGetCommand creates a new `qri search` command that searches for datasets
func NewGetCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &GetOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "get [COMPONENT] [DATASET]",
		Short: "get components of qri datasets",
		Long: `Get the qri dataset (except for the body). You can also get components of 
the dataset: meta, structure, viz, transform, and commit. To narrow down
further to specific fields in each section, use dot notation. The get 
command prints to the console in yaml format, by default.

Check out https://qri.io/docs/reference/dataset/ to learn about each section of the 
dataset and its fields.`,
		Example: `  # Print the entire dataset to the console:
  $ qri get me/annual_pop

  # Print the meta to the console:
  $ qri get meta me/annual_pop

  # Print the dataset body size to the console:
  $ qri get structure.length me/annual_pop`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Special case for --pretty, check if it was passed vs if the default was used.
			if cmd.Flags().Changed("pretty") {
				o.HasPretty = true
			}
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json, yaml]")
	cmd.Flags().BoolVar(&o.Pretty, "pretty", false, "whether to print output with indentation, only for json format")
	cmd.Flags().IntVar(&o.PageSize, "page-size", -1, "for body, limit how many entries to get per page")
	cmd.Flags().IntVar(&o.Page, "page", -1, "for body, page at which to get entries")
	cmd.Flags().BoolVarP(&o.All, "all", "a", true, "for body, whether to get all entries")
	cmd.Flags().StringVarP(&o.Outfile, "outfile", "o", "", "file to write output to")

	return cmd
}

// GetOptions encapsulates state for the get command
type GetOptions struct {
	ioes.IOStreams

	Refs     *RefSelect
	Selector string
	Format   string

	Page     int
	PageSize int
	All      bool

	Pretty    bool
	HasPretty bool
	Outfile   string

	DatasetMethods *lib.DatasetMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *GetOptions) Complete(f Factory, args []string) (err error) {
	if o.DatasetMethods, err = f.DatasetMethods(); err != nil {
		return
	}

	if len(args) > 0 {
		if component.IsDatasetField.MatchString(args[0]) {
			o.Selector = args[0]
			args = args[1:]
		}
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, AnyNumberOfReferences, nil); err != nil {
		return
	}

	if o.Selector == "body" {
		// if we have a PageSize, but not Page, assume an Page of 1
		if o.PageSize != -1 && o.Page == -1 {
			o.Page = 1
		}
		// set all to false if PageSize or Page values are provided
		if o.PageSize != -1 || o.Page != -1 {
			o.All = false
		}
	} else {
		if o.PageSize != -1 {
			return fmt.Errorf("can only use --page-size flag when getting body")
		}
		if o.Page != -1 {
			return fmt.Errorf("can only use --page flag when getting body")
		}
		if !o.All {
			return fmt.Errorf("can only use --all flag when getting body")
		}
	}

	return nil
}

// Run executes the get command
func (o *GetOptions) Run() (err error) {
	printRefSelect(o.ErrOut, o.Refs)

	// Pretty maps to a key in the FormatConfig map.
	var fc dataset.FormatConfig
	if o.HasPretty {
		opt := dataset.JSONOptions{Options: make(map[string]interface{})}
		opt.Options["pretty"] = o.Pretty
		fc = &opt
	}

	// TODO(dustmop): Consider setting o.Format if o.Outfile has an extension and o.Format
	// is not assigned anything

	// TODO(dustmop): Allow any number of references. Right now we ignore everything after the
	// first. The hard parts are:
	// 1) Correctly handling the pager output, and having headers between each ref
	// 2) Identifying cases that limit Get to only work on 1 dataset. For example, the -o flag

	// convert Page and PageSize to Limit and Offset
	page := util.NewPage(o.Page, o.PageSize)
	p := lib.GetParams{
		Refstr:       o.Refs.Ref(),
		Selector:     o.Selector,
		Format:       o.Format,
		FormatConfig: fc,
		Offset:       page.Offset(),
		Limit:        page.Limit(),
		All:          o.All,
		Outfile:      o.Outfile,
		// Generate a filename only if we're outputting to a terminal (not a pipe), and we're
		// outputting a zip. lib.Get will also check that we're outputting a zip, this check is
		// repeated here for clarity.
		GenFilename: o.Outfile == "" && stdoutIsTerminal() && o.Format == "zip",
	}
	res := lib.GetResult{}
	if err = o.DatasetMethods.Get(&p, &res); err != nil {
		return err
	}
	if res.Message != "" {
		o.Out.Write([]byte(res.Message))
		o.Out.Write([]byte{'\n'})
		return nil
	}
	if len(res.Bytes) > 0 {
		buf := bytes.NewBuffer(res.Bytes)
		buf.Write([]byte{'\n'})
		printToPager(o.Out, buf)
	}
	return nil
}
