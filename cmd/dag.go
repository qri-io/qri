package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/ghodss/yaml"
	"github.com/qri-io/dag"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base/component"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewDAGCommand creates a new `qri dag` command that generates a manifest for a given
// dataset reference. Referenced dataset must be stored in local CAFS
func NewDAGCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &DAGOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:    "dag",
		Hidden: true,
		Short:  "directed acyclic graph interaction",
	}

	manifest := &cobra.Command{
		Use:   "manifest",
		Short: "dataset manifest interaction",
	}

	get := &cobra.Command{
		Use:   "get DATASET [DATASET...]",
		Short: "get manifests for one or more dataset references",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args, false); err != nil {
				return err
			}
			return o.Get()
		},
	}

	missing := &cobra.Command{
		Use:   "missing --file MANIFEST_PATH",
		Short: "list blocks not present in this repo for a given manifest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args, false); err != nil {
				return err
			}
			return o.Missing()
		},
	}

	get.Flags().StringVar(&o.Format, "format", "json", "set output format [json, yaml, cbor]")
	get.Flags().BoolVar(&o.Pretty, "pretty", false, "print output without indentation, only applies to json format")
	get.Flags().BoolVar(&o.Hex, "hex", false, "hex-encode output")

	missing.Flags().StringVar(&o.Format, "format", "json", "set output format [json, yaml, cbor]")
	missing.Flags().BoolVar(&o.Pretty, "pretty", false, "print output without indentation, only applies to json format")
	missing.Flags().BoolVar(&o.Hex, "hex", false, "hex-encode output")
	missing.Flags().StringVar(&o.File, "file", "", "manifest file")
	missing.MarkFlagRequired("file")

	manifest.AddCommand(get, missing)

	info := &cobra.Command{
		Use:   "info [LABEL] DATASET [DATASET...]",
		Short: "dataset dag info interaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args, true); err != nil {
				return err
			}
			return o.Info()
		},
	}

	info.Flags().StringVar(&o.InfoFormat, "format", "", "set output format [json, yaml, cbor]")
	info.Flags().BoolVar(&o.Pretty, "pretty", false, "print output without indentation, only applies to json format")
	info.Flags().BoolVar(&o.Hex, "hex", false, "hex-encode output")

	cmd.AddCommand(manifest, info)
	return cmd
}

// DAGOptions encapsulates state for the dag command
type DAGOptions struct {
	ioes.IOStreams

	Refs       []string
	Format     string
	InfoFormat string
	Pretty     bool
	Hex        bool
	File       string
	Label      string

	inst *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *DAGOptions) Complete(f Factory, args []string, parseLabel bool) (err error) {
	if parseLabel && len(args) > 0 {
		if component.IsDatasetField.MatchString(args[0]) {
			o.Label = fullFieldToAbbr(args[0])
			args = args[1:]
		}
	}
	o.Refs = args
	o.inst, err = f.Instance()
	return
}

// Get executes the manifest get command
func (o *DAGOptions) Get() (err error) {
	ctx := context.TODO()
	mf := &dag.Manifest{}
	for _, ref := range o.Refs {
		if mf, err = o.inst.Dataset().Manifest(ctx, &lib.ManifestParams{Ref: ref}); err != nil {
			return err
		}

		var buffer []byte
		switch strings.ToLower(o.Format) {
		case "json":
			if !o.Pretty {
				buffer, err = json.Marshal(mf)
			} else {
				buffer, err = json.MarshalIndent(mf, "", " ")
			}
		case "yaml":
			buffer, err = yaml.Marshal(mf)
		case "cbor":
			buffer, err = mf.MarshalCBOR()
		}
		if err != nil {
			return fmt.Errorf("err encoding manifest: %s", err)
		}
		if o.Hex {
			buffer = []byte(hex.EncodeToString(buffer))
		}
		_, err = o.Out.Write(buffer)
	}

	return err
}

// Missing executes the manifest missing command
func (o *DAGOptions) Missing() error {
	ctx := context.TODO()
	in := &dag.Manifest{}
	data, err := ioutil.ReadFile(o.File)
	if err != nil {
		return err
	}

	switch strings.ToLower(filepath.Ext(o.File)) {
	case ".yaml":
		err = yaml.Unmarshal(data, in)
	case ".json":
		err = json.Unmarshal(data, in)
	case ".cbor":
		// TODO - detect hex input?
		// data, err = hex.DecodeString(string(data))
		// if err != nil {
		// 	return err
		// }
		in, err = dag.UnmarshalCBORManifest(data)
	}

	if err != nil {
		return err
	}

	mf, err := o.inst.Dataset().ManifestMissing(ctx, &lib.ManifestMissingParams{Manifest: in})
	if err != nil {
		return err
	}

	var buffer []byte
	switch strings.ToLower(o.InfoFormat) {
	case "json":
		if !o.Pretty {
			buffer, err = json.Marshal(mf)
		} else {
			buffer, err = json.MarshalIndent(mf, "", " ")
		}
	case "yaml":
		buffer, err = yaml.Marshal(mf)
	case "cbor":
		buffer, err = mf.MarshalCBOR()
	}
	if err != nil {
		return fmt.Errorf("error encoding manifest: %s", err)
	}
	if o.Hex {
		buffer = []byte(hex.EncodeToString(buffer))
	}
	_, err = o.Out.Write(buffer)

	return err
}

// Info executes the dag info command
func (o *DAGOptions) Info() (err error) {
	ctx := context.TODO()
	info := &dag.Info{}
	if len(o.Refs) == 0 {
		return fmt.Errorf("dataset reference required")
	}

	for _, ref := range o.Refs {
		s := &lib.DAGInfoParams{Ref: ref, Label: o.Label}
		info, err = o.inst.Dataset().DAGInfo(ctx, s)
		if err != nil {
			return err
		}

		var buffer []byte
		switch strings.ToLower(o.InfoFormat) {
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
				out += fmt.Sprintf("\nSubDAG at: %s", abbrFieldToFull(o.Label))
			}
			out += fmt.Sprintf("\nDAG for: %s\n", ref)
			if totalSize != 0 {
				out += fmt.Sprintf("Total Size: %s\n", humanize.Bytes(totalSize))
			}
			if info.Manifest != nil {
				out += fmt.Sprintf("Block Count: %d\n", len(info.Manifest.Nodes))
			}
			if info.Labels != nil {
				out += fmt.Sprint("Labels:\n")
			}
			for label, index := range info.Labels {
				fullField := abbrFieldToFull(label)
				out += fmt.Sprintf("\t%s:", fullField)
				spacesLen := 16 - len(fullField)
				for i := 0; i <= spacesLen; i++ {
					out += fmt.Sprintf(" ")
				}
				out += fmt.Sprintf("%s\n", humanize.Bytes(info.Sizes[index]))
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

func fullFieldToAbbr(field string) string {
	switch field {
	case "commit":
		return "cm"
	case "structure":
		return "st"
	case "body":
		return "bd"
	case "meta":
		return "md"
	case "viz":
		return "vz"
	case "transform":
		return "tf"
	case "rendered":
		return "rd"
	default:
		return field
	}
}

func abbrFieldToFull(abbr string) string {
	switch abbr {
	case "cm":
		return "commit"
	case "st":
		return "structure"
	case "bd":
		return "body"
	case "md":
		return "meta"
	case "vz":
		return "viz"
	case "tf":
		return "transform"
	case "rd":
		return "rendered"
	default:
		return abbr
	}
}
