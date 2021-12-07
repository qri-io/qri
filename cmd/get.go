package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
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
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json, yaml, csv, zip]. If format is set to 'zip' it will save the entire dataset as a zip archive.")
	cmd.Flags().BoolVar(&o.Pretty, "pretty", false, "whether to print output with indentation, only for json format")
	cmd.Flags().IntVar(&o.Limit, "limit", -1, "for body, limit how many entries to get per request")
	cmd.Flags().IntVar(&o.Offset, "offset", -1, "for body, offset amount at which to get entries")
	cmd.Flags().BoolVarP(&o.All, "all", "a", true, "for body, whether to get all entries")
	cmd.Flags().StringVarP(&o.Outfile, "outfile", "o", "", "file to write output to")

	cmd.Flags().BoolVar(&o.Offline, "offline", false, "prevent network access")
	cmd.Flags().StringVar(&o.Remote, "remote", "", "name to get any remote data from")

	return cmd
}

// GetOptions encapsulates state for the get command
type GetOptions struct {
	ioes.IOStreams

	Refs     *RefSelect
	Selector string
	Format   string

	Limit  int
	Offset int
	All    bool

	Pretty  bool
	Outfile string

	Offline bool
	Remote  string

	inst *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *GetOptions) Complete(f Factory, args []string) (err error) {
	if o.inst, err = f.Instance(); err != nil {
		return
	}

	if len(args) > 0 {
		if component.IsDatasetField.MatchString(args[0]) {
			o.Selector = args[0]
			args = args[1:]
		}
	}
	if o.Refs, err = GetCurrentRefSelect(f, args, AnyNumberOfReferences); err != nil {
		return
	}

	if o.Selector == "body" {
		if o.Limit != -1 && o.Offset == -1 {
			o.Offset = 0
		}
		if o.Offset != -1 || o.Limit != -1 {
			o.All = false
		}
	} else {
		if o.Format == "csv" {
			return fmt.Errorf("can only use --format=csv when getting body")
		}
		if o.Limit != -1 {
			return fmt.Errorf("can only use --limit flag when getting body")
		}
		if o.Offset != -1 {
			return fmt.Errorf("can only use --offset flag when getting body")
		}
		if !o.All {
			return fmt.Errorf("can only use --all flag when getting body")
		}
	}

	return
}

// Run executes the get command
func (o *GetOptions) Run() (err error) {
	if o.Offline {
		if o.Remote != "" {
			return fmt.Errorf("cannot use '--offline' and '--remote' flags together")
		}
		o.Remote = "local"
	}

	// TODO(dustmop): Allow any number of references. Right now we ignore everything after the
	// first. The hard parts are:
	// 1) Correctly handling the pager output, and having headers between each ref
	// 2) Identifying cases that limit Get to only work on 1 dataset. For example, the -o flag

	ctx := context.TODO()
	p := &lib.GetParams{
		Ref:      o.Refs.Ref(),
		Selector: o.Selector,
		Offset:   o.Offset,
		Limit:    o.Limit,
		All:      o.All,
	}
	var outBytes []byte
	switch {
	case o.Format == "zip":
		zipResults, err := o.inst.Dataset().GetZip(ctx, p)
		if err != nil {
			return err
		}
		outBytes = zipResults.Bytes
		if o.Outfile == "" {
			o.Outfile = zipResults.GeneratedName
		}
	case o.Format == "csv":
		outBytes, err = o.inst.Dataset().GetCSV(ctx, p)
		if err != nil {
			return err
		}
	default:
		res, err := o.inst.WithSource(o.Remote).Dataset().Get(ctx, p)
		if err != nil {
			return err
		}
		switch {
		case lib.IsSelectorScriptFile(o.Selector):
			outBytes = res.Bytes
		case o.Format == "json" || (o.Selector == "body" && o.Format == ""):
			if o.Pretty {
				outBytes, err = json.MarshalIndent(res.Value, "", "  ")
				if err != nil {
					return err
				}
				break
			}

			outBytes, err = json.Marshal(res.Value)
			if err != nil {
				return err
			}
		default:
			outBytes, err = yaml.Marshal(res.Value)
			if err != nil {
				return err
			}
		}
	}

	if o.Outfile != "" {
		err := ioutil.WriteFile(o.Outfile, outBytes, 0644)
		if err != nil {
			return err
		}
		outBytes = []byte(fmt.Sprintf("wrote to file %q", o.Outfile))
	}
	buf := bytes.NewBuffer(outBytes)
	buf.Write([]byte{'\n'})
	printToPager(o.Out, buf)
	return nil
}
