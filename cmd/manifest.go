package cmd

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
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
		Short:  "dataset manifest interation",
	}

	get := &cobra.Command{
		Use:   "get",
		Short: "get one or more manifests for a given reference",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Get()
		},
	}

	missing := &cobra.Command{
		Use:   "missing",
		Short: "list blocks not present in this repo for a given manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Missing()
		},
	}

	get.Flags().StringVar(&o.Format, "format", "json", "set output format [json, yaml, cbor]")
	get.Flags().BoolVar(&o.Pretty, "pretty", false, "print output without indentation, only applies to json format")

	missing.Flags().StringVar(&o.Format, "format", "json", "set output format [json, yaml, cbor]")
	missing.Flags().BoolVar(&o.Pretty, "pretty", false, "print output without indentation, only applies to json format")
	missing.Flags().StringVar(&o.File, "file", "", "manifest file")

	cmd.AddCommand(get, missing)

	return cmd
}

// ManifestOptions encapsulates state for the get command
type ManifestOptions struct {
	ioes.IOStreams

	Refs   []string
	Format string
	Pretty bool
	File   string

	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ManifestOptions) Complete(f Factory, args []string) (err error) {
	o.Refs = args
	o.DatasetRequests, err = f.DatasetRequests()
	return
}

// Get executes the get command
func (o *ManifestOptions) Get() (err error) {
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

// Missing executes the missing command
func (o *ManifestOptions) Missing() error {
	if o.File == "" {
		return fmt.Errorf("manifest file is required")
	}

	in := &manifest.Manifest{}
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
		// TODO - I'm not a fan of this hex tom-foolery
		data, err = hex.DecodeString(string(data))
		if err != nil {
			return err
		}
		in, err = manifest.UnmarshalCBOR(data)
	}

	if err != nil {
		return err
	}

	mf := &manifest.Manifest{}
	if err = o.DatasetRequests.ManifestMissing(in, mf); err != nil {
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

	return err
}
