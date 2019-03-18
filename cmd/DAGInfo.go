package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dag"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewDAGInfoCommand creates a new `qri daginfo` command that generates a daginfo for a given
// dataset reference. Referenced dataset must be stored in local CAFS
func NewDAGInfoCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &DAGInfoOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:    "daginfo",
		Hidden: true,
		Short:  "dataset daginfo interaction",
	}

	get := &cobra.Command{
		Use:   "get",
		Short: "get one or more DAG info for a given reference",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Get()
		},
	}

	// missing := &cobra.Command{
	// 	Use:   "missing",
	// 	Short: "list blocks not present in this repo for a given daginfo",
	// 	RunE: func(cmd *cobra.Command, args []string) error {
	// 		if err := o.Complete(f, args); err != nil {
	// 			return err
	// 		}
	// 		return o.Missing()
	// 	},
	// }

	get.Flags().StringVar(&o.Format, "format", "", "set output format [json, yaml, cbor]")
	get.Flags().BoolVar(&o.Pretty, "pretty", false, "print output without indentation, only applies to json format")
	get.Flags().BoolVar(&o.Hex, "hex", false, "hex-encode output")

	// missing.Flags().StringVar(&o.Format, "format", "json", "set output format [json, yaml, cbor]")
	// missing.Flags().BoolVar(&o.Pretty, "pretty", false, "print output without indentation, only applies to json format")
	// missing.Flags().BoolVar(&o.Hex, "hex", false, "hex-encode output")
	// missing.Flags().StringVar(&o.File, "file", "", "daginfo file")

	cmd.AddCommand(get)
	// cmd.AddCommand(get, missing)

	return cmd
}

// DAGInfoOptions encapsulates state for the daginfo command
type DAGInfoOptions struct {
	ioes.IOStreams

	Refs   []string
	Format string
	Pretty bool
	Hex    bool
	File   string
	Label  string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *DAGInfoOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		if isDatasetField.MatchString(args[0]) {
			o.Label = args[0]
			args = args[1:]
		}
	}
	o.Refs = args
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Get executes the get command
func (o *DAGInfoOptions) Get() (err error) {
	info := &dag.Info{}
	for _, refstr := range o.Refs {
		s := &lib.DAGInfoParams{RefStr: refstr, Label: o.Label}
		if err = o.DatasetRequests.DAGInfo(s, info); err != nil {
			return err
		}

		var buffer []byte
		switch strings.ToLower(o.Format) {
		case "json":
			if !o.Pretty {
				buffer, err = json.Marshal(info)
			} else {
				buffer, err = json.MarshalIndent(info, "", " ")
			}
		case "yaml":
			buffer, err = yaml.Marshal(info)
		// case "cbor":
		// 	buffer, err = info.MarshalCBOR()
		default:
			totalSize := uint64(0)
			if len(info.Sizes) != 0 {
				totalSize = info.Sizes[0]
			}
			out := ""
			if o.Label != "" {
				out += fmt.Sprintf("\nSubDAG at: %s", o.Label)
			}
			out += fmt.Sprintf("\nDAG for: %s\n", refstr)
			if totalSize != 0 {
				out += fmt.Sprintf("Total Size: %s\n", printByteInfo(int(totalSize)))
			}
			if info.Manifest != nil {
				out += fmt.Sprintf("Block Count: %d\n", len(info.Manifest.Nodes))
			}
			if info.Labels != nil {
				out += fmt.Sprint("Labels:\n")
			}
			for label, index := range info.Labels {
				out += fmt.Sprintf("\t%s:", label)
				spacesLen := 16 - len(label)
				for i := 0; i <= spacesLen; i++ {
					out += fmt.Sprintf(" ")
				}
				out += fmt.Sprintf("%s\n", printByteInfo(int(info.Sizes[index])))
			}
			buffer = []byte(out)

		}
		if err != nil {
			return fmt.Errorf("err encoding daginfo: %s", err)
		}
		if o.Hex {
			buffer = []byte(hex.EncodeToString(buffer))
		}
		_, err = o.Out.Write(buffer)
	}

	return err
}
