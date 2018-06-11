package cmd

import (
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
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/spf13/cobra"
)

// NewExportCommand creates a new export cobra command
// exportCmd represents the export command
func NewExportCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &ExportOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "copy datasets to your local filesystem",
		Long: `
Export gets datasets out of qri. By default it exports only a dataset’s data to 
the path [current directory]/[peername]/[dataset name]/[data file]. 

To export everything about a dataset, use the --dataset flag.`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.Complete(f, args))
			ExitIfErr(o.Run())
		},
	}

	cmd.Flags().BoolVarP(&o.Blank, "blank", "", false, "export a blank dataset YAML file, overrides all other flags except output")
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "path to write to, default is current directory")
	cmd.Flags().StringVarP(&o.Format, "format", "f", "yaml", "format for all exported files, except for body. yaml is the default format. options: yaml, json")
	cmd.Flags().StringVarP(&o.BodyFormat, "body-format", "", "", "format for dataset body. default is the original data format. options: json, csv, cbor")
	cmd.Flags().BoolVarP(&o.Zipped, "zip", "z", false, "compress export as zip archive, export all parts of dataset, data in original format")
	cmd.Flags().BoolVarP(&o.All, "all", "a", false, "export full dataset package")
	cmd.Flags().BoolVarP(&o.Namespaced, "namespaced", "n", false, "export to a peer name namespaced directory")
	cmd.Flags().BoolVarP(&o.Dataset, "dataset", "", false, "export root dataset")
	cmd.Flags().BoolVarP(&o.Meta, "meta", "m", false, "export dataset metadata file")
	cmd.Flags().BoolVarP(&o.Structure, "structure", "s", false, "export dataset structure file")
	cmd.Flags().BoolVarP(&o.Transform, "transform", "t", false, "export dataset transformation file & details")
	cmd.Flags().BoolVarP(&o.NoBody, "no-body", "", false, "don't include dataset body in export")
	// exportCmd.Flags().BoolVarP(&exportCmdVis, "vis-conf", "c", false, "export viz config file")

	return cmd
}

// ExportOptions encapsulates state for the export command
type ExportOptions struct {
	IOStreams

	Ref        string
	Dataset    bool
	Meta       bool
	Structure  bool
	NoBody     bool
	Transform  bool
	Vis        bool
	All        bool
	Namespaced bool
	Zipped     bool
	Blank      bool
	Output     string
	Format     string
	BodyFormat string

	UsingRPC        bool
	Repo            repo.Repo
	Profile         *profile.Profile
	DatasetRequests *core.DatasetRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *ExportOptions) Complete(f Factory, args []string) (err error) {
	if len(args) > 0 {
		o.Ref = args[0]
	}
	o.UsingRPC = f.RPC() != nil
	o.DatasetRequests, err = f.DatasetRequests()
	o.Repo, err = f.Repo()
	if err != nil {
		return
	}

	o.Profile, err = o.Repo.Profile()
	return err
}

// Run executes the Export command
func (o *ExportOptions) Run() error {
	if o.UsingRPC {
		return usingRPCError("export")
	}

	path := o.Output
	format := o.Format
	bodyFormat := o.BodyFormat
	if bodyFormat != "" && !(bodyFormat == "json" || bodyFormat == "csv" || bodyFormat == "cbor") {
		ErrExit(fmt.Errorf("%s is not an accepted data format, options are json, csv, and cbor", bodyFormat))
	}

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

	dsr, err := repo.ParseDatasetRef(o.Ref)
	if err != nil {
		return err
	}

	res := &repo.DatasetRef{}
	if err = o.DatasetRequests.Get(&dsr, res); err != nil {
		return err
	}

	ds, err := res.DecodeDataset()
	if err != nil {
		return err
	}

	if o.Namespaced {
		peerName := dsr.Peername
		if peerName == "me" {
			peerName = o.Profile.Peername
		}
		path = filepath.Join(path, peerName)
	}
	path = filepath.Join(path, dsr.Name)

	if o.Zipped {
		dst, err := os.Create(fmt.Sprintf("%s.zip", path))
		if err != nil {
			return err
		}

		if err = dsutil.WriteZipArchive(o.Repo.Store(), ds, dst); err != nil {
			return err
		}
		return dst.Close()
	} else if o.All {
		o.NoBody = false
		o.Dataset = true
		o.Meta = true
		o.Structure = true
		o.Transform = true
	}

	if path != "" {
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}

	if o.Meta {
		var md interface{}
		// TODO - this ensures a "form" metadata file is written
		// when one doesn't exist. This should be better
		if ds.Meta != nil && ds.Meta.IsEmpty() {
			md = struct {
				AccessPath         string       `json:"accessPath"`
				AccrualPeriodicity string       `json:"accrualPeriodicity"`
				Citations          []string     `json:"citations"`
				Description        string       `json:"description"`
				DownloadPath       string       `json:"downloadPath"`
				HomePath           string       `json:"homePath"`
				Identifier         string       `json:"identifier"`
				Keywords           []string     `json:"keywords"`
				Language           []string     `json:"language"`
				License            string       `json:"license"`
				Qri                dataset.Kind `json:"qri"`
				ReadmePath         string       `json:"readmePath"`
				Title              string       `json:"title"`
				Theme              []string     `json:"theme"`
				Version            string       `json:"version"`
			}{
				Qri: dataset.KindMeta,
			}
		} else {
			md = ds.Meta
		}

		metaPath := filepath.Join(path, dsfs.PackageFileMeta.Filename())
		var mdBytes []byte

		switch format {
		case "json":
			mdBytes, err = json.MarshalIndent(md, "", "  ")
			if err != nil {
				return err
			}
		default:
			mdBytes, err = yaml.Marshal(md)
			if err != nil {
				return err
			}
			metaPath = fmt.Sprintf("%s.yaml", strings.TrimSuffix(metaPath, filepath.Ext(metaPath)))
		}
		if err = ioutil.WriteFile(metaPath, mdBytes, os.ModePerm); err != nil {
			return err
		}
		printSuccess(o.Out, "exported metadata file to: %s", metaPath)
	}

	if o.Structure {
		stPath := filepath.Join(path, dsfs.PackageFileStructure.Filename())
		var stBytes []byte

		switch format {
		case "json":
			stBytes, err = json.MarshalIndent(ds.Structure, "", "  ")
			if err != nil {
				return err
			}
		default:
			stBytes, err = yaml.Marshal(ds.Structure)
			if err != nil {
				return err
			}
			stPath = fmt.Sprintf("%s.yaml", strings.TrimSuffix(stPath, filepath.Ext(stPath)))
		}
		if err = ioutil.WriteFile(stPath, stBytes, os.ModePerm); err != nil {
			return err
		}
		printSuccess(o.Out, "exported structure file to: %s", stPath)
	}

	writeTransformScript := func() error {
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
		return nil
	}

	if o.Transform {
		tfPath := filepath.Join(path, dsfs.PackageFileTransform.Filename())
		var stBytes []byte

		switch format {
		case "json":
			stBytes, err = json.MarshalIndent(ds.Transform, "", "  ")
			if err != nil {
				return err
			}
		default:
			stBytes, err = yaml.Marshal(ds.Transform)
			if err != nil {
				return err
			}
			tfPath = fmt.Sprintf("%s.yaml", strings.TrimSuffix(tfPath, filepath.Ext(tfPath)))
		}
		if err = ioutil.WriteFile(tfPath, stBytes, os.ModePerm); err != nil {
			return err
		}
		printSuccess(o.Out, "exported transform file to: %s", tfPath)
		if err = writeTransformScript(); err != nil {
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

		p := &core.LookupParams{
			Format: df,
			Path:   ds.Path().String(),
			All:    true,
		}
		r := &core.LookupResult{}

		if err = o.DatasetRequests.LookupBody(p, r); err != nil {
			return err
		}

		dataPath := filepath.Join(path, fmt.Sprintf("data.%s", bodyFormat))
		dst, err := os.Create(dataPath)
		if err != nil {
			return err
		}

		if _, err = dst.Write(r.Data); err != nil {
			return err
		}

		if err = dst.Close(); err != nil {
			return err
		}
		printSuccess(o.Out, "exported data to: %s", dataPath)
	}

	if o.Dataset {
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
		if err = writeTransformScript(); err != nil {
			return err
		}

		printSuccess(o.Out, "exported dataset.json to: %s", dsPath)
	}

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
  # tags:

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
# that qri can execute for you. Currently transforms are written in the skylark
# scripting language, which is modeled after the python programming language
# for more info check https://qri.io/docs/transforms
# transform:
#   scriptpath: tf.sky

# data itself is either a path to a file on your computer,
# or a URL that leads to the raw data
# dataPath:
`
