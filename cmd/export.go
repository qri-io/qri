package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/spf13/cobra"
)

// NewExportCommand creates a new export cobra command
// exportCmd represents the export command
func NewExportCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &ExportOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Copy datasets to your local filesystem",
		Long: `
Export gets datasets out of qri. By default it exports the dataset body, as ` + "`body.csv`" + `, header as` + "`dataset.json`" + `, and ref, as ` + "`ref.txt`" + ` files. 

To export to a specific directory, use the --output flag.

If you want an empty dataset that can be filled in with details to create a
new dataset, use --blank.`,
		Example: `  # export dataset
  qri export me/annual_pop

  # export without the body of the dataset
  qri export --no-body me/annual_pop

  # export to a specific directory
  qri export -o ~/new_directory me/annual_pop`,
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

	cmd.Flags().BoolVarP(&o.Blank, "blank", "", false, "export a blank dataset YAML file, overrides all other flags except output")
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "path to write to, default is current directory")
	cmd.Flags().StringVarP(&o.Format, "format", "f", "yaml", "format for all exported files, except for body. yaml is the default format. options: yaml, json")
	cmd.Flags().StringVarP(&o.BodyFormat, "body-format", "", "", "format for dataset body. default is the original data format. options: json, csv, cbor")
	cmd.Flags().BoolVarP(&o.NoBody, "no-body", "b", false, "don't include dataset body in export")
	cmd.Flags().BoolVarP(&o.PeerDir, "peer-dir", "d", false, "export to a peer name namespaced directory")

	return cmd
}

// ExportOptions encapsulates state for the export command
type ExportOptions struct {
	ioes.IOStreams

	Ref        string
	PeerDir    bool
	Zipped     bool
	Blank      bool
	Output     string
	Format     string
	BodyFormat string
	NoBody     bool

	UsingRPC        bool
	Repo            repo.Repo
	Profile         *profile.Profile
	DatasetRequests *lib.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ExportOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		o.Ref = args[0]
	}
	if f.RPC() != nil {
		return usingRPCError("export")
	}

	o.DatasetRequests, err = f.DatasetRequests()
	// TODO (dlong): All other callers of this method have removed their need of it. Remove
	// this call and then remove it from the factory interface.
	o.Repo, err = f.Repo()
	if err != nil {
		return
	}

	o.Profile, err = o.Repo.Profile()
	return err
}

// Run executes the Export command
func (o *ExportOptions) Run() error {
	path := o.Output
	format := o.Format
	bodyFormat := o.BodyFormat

	if o.Blank {
		if path == "" {
			path = "dataset.yaml"
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := ioutil.WriteFile(path, []byte(blankYamlDataset), os.ModePerm); err != nil {
				return err
			}
			printSuccess(o.Out, "blank dataset file saved to %s", path)
			return nil
		}
		return fmt.Errorf("'%s' already exists", path)
	}

	if bodyFormat != "" && !(bodyFormat == "json" || bodyFormat == "csv" || bodyFormat == "cbor") {
		return fmt.Errorf("%s is not an accepted data format, options are json, csv, and cbor", bodyFormat)
	}

	dsr, err := repo.ParseDatasetRef(o.Ref)
	if err != nil && err != repo.ErrEmptyRef {
		return err
	}

	res := &repo.DatasetRef{}
	if err = o.DatasetRequests.Get(&dsr, res); err != nil {
		if err == repo.ErrEmptyRef {
			return lib.NewError(err, "please provide a dataset reference")
		}
		return err
	}

	ds, err := res.DecodeDataset()
	if err != nil {
		return err
	}

	if o.PeerDir {
		peerName := dsr.Peername
		if peerName == "me" {
			peerName = o.Profile.Peername
		}
		path = filepath.Join(path, peerName)
	}
	path = filepath.Join(path, dsr.Name)

	// TODO: Implement flags specified in the RFC 0014, only set the zip flag if the export
	// includes multiple files and the --directories-for-files flag is not true.
	o.Zipped = true
	if o.Zipped {
		dst, err := os.Create(fmt.Sprintf("%s.zip", path))
		if err != nil {
			return err
		}

		if err = dsutil.WriteZipArchive(o.Repo.Store(), ds, res.String(), dst); err != nil {
			return err
		}
		return dst.Close()
	}

	if path != "" {
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}

	if !o.NoBody {
		if bodyFormat == "" {
			bodyFormat = ds.Structure.Format.String()
		}

		df, err := dataset.ParseDataFormatString(bodyFormat)
		if err != nil {
			return err
		}

		p := &lib.LookupParams{
			Format: df,
			Path:   ds.Path().String(),
			All:    true,
		}
		r := &lib.LookupResult{}

		if err = o.DatasetRequests.LookupBody(p, r); err != nil {
			return err
		}

		dataPath := filepath.Join(path, fmt.Sprintf("data.%s", bodyFormat))
		dst, err := os.Create(dataPath)
		if err != nil {
			return err
		}

		if p.Format == dataset.CBORDataFormat {
			r.Data = []byte(hex.EncodeToString(r.Data))
		}
		if _, err = dst.Write(r.Data); err != nil {
			return err
		}

		if err = dst.Close(); err != nil {
			return err
		}
		printSuccess(o.Out, "exported data to: %s", dataPath)
	}

	dsPath := filepath.Join(path, dsfs.PackageFileDataset.String())
	var dsBytes []byte

	switch format {
	case "json":
		dsBytes, err = json.MarshalIndent(ds, "", "  ")
		if err != nil {
			return err
		}
	default:
		dsBytes, err = yaml.Marshal(ds)
		if err != nil {
			return err
		}
		dsPath = fmt.Sprintf("%s.yaml", strings.TrimSuffix(dsPath, filepath.Ext(dsPath)))
	}
	if err = ioutil.WriteFile(dsPath, dsBytes, os.ModePerm); err != nil {
		return err
	}

	if ds.Transform != nil && ds.Transform.ScriptPath != "" {
		f, err := o.Repo.Store().Get(datastore.NewKey(ds.Transform.ScriptPath))
		if err != nil {
			return err
		}
		scriptData, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		// TODO - transformations should have default file extensions
		if err = ioutil.WriteFile(filepath.Join(path, "transform.sky"), scriptData, os.ModePerm); err != nil {
			return err
		}
		printSuccess(o.Out, "exported transform script to: %s", filepath.Join(path, "transform.sky"))
	}

	printSuccess(o.Out, "exported dataset.json to: %s", dsPath)

	return nil
}

const blankYamlDataset = `# This file defines a qri dataset. Change this file, save it, then from a terminal run:
# $ qri add --file=dataset.yaml
# For more info check out https://qri.io/docs

# Name is a short name for working with this dataset without spaces for example:
# "my_dataset" or "number_of_cows_that_have_jumped_the_moon"
# name is required
name: 

# Commit contains notes about this dataset at the time it was saved
# all commit stuff is optional (one will be generated for you if you don't provide one)
commit:
  title:
  message:

# Meta stores descriptive information about a dataset.
# all meta info is optional, but you should at least add a title.
# detailed, accurate metadata helps you & others find your data later.
meta:
  title:
  # description:
  # category:
  # keywords:

# Structure contains the info a computer needs to interpret this dataset
# qri will figure structure out for you if you don't one
# and later you can change structure to do neat stuff like validate your
# data and make your data work with other datasets.
# Below is an example structure
structure:
  # Syntax in JSON format:
  # format: json
  # Schema defines the "shape" data should take, here we're saying
  # data should be an array of strings, like this: ["foo", "bar", "baz"]
  # schema:
  #   type: array
  #   items:
  #     type: string

# Transform contains instructions for creating repeatable, auditable scripts
# that qri can execute for you. Currently transforms are written in the starlark
# scripting language, which is modeled after the python programming language
# for more info check https://qri.io/docs/transforms
# transform:
#   scriptpath: tf.sky

# use viz to provide custom a HTML template of your dataset
# the currently accepted syntax is 'html'
# scriptpath is the path to your template, relative to this file:
# viz:
#   syntax: html
#   scriptpath: template.html

# the body of a dataset is data itself. either a path to a file on your computer,
# or a URL that leads to the raw data
# bodypath:
`
