package cmd

import (
	"fmt"
	"strings"

	"encoding/hex"
	"encoding/json"

	"github.com/ghodss/yaml"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/manifest"
	"github.com/spf13/cobra"
)

// NewManifestCommand creates a new `qri search` command that searches for datasets
func NewManifestCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ManifestOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:    "manifest",
		Hidden: true,
		Short:  "generate a qri manifest",
		// Annotations: map[string]string{
		// 	"group": "dataset",
		// },
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Format, "format", "f", "json", "set output format [json, yaml, cbor]")
	cmd.Flags().BoolVar(&o.Pretty, "pretty", false, "print output without indentation, only applies to json format")

	return cmd
}

// ManifestOptions encapsulates state for the get command
type ManifestOptions struct {
	ioes.IOStreams

	Refs   []string
	Format string
	Pretty bool

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ManifestOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Run executes the get command
func (o *ManifestOptions) Run() (err error) {
	mf := &manifest.Manifest{}
	for _, refstr := range o.Refs {
		if err = o.DatasetRequests.Manifest(&refstr, mf); err != nil {
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
			var raw []byte
			raw, err = mf.MarshalCBOR()
			buffer = []byte(hex.EncodeToString(raw))
		}
		if err != nil {
			return fmt.Errorf("error getting config: %s", err)
		}
		_, err = o.Out.Write(buffer)
	}

	return err
}
