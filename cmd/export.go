package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	exportCmdDataset    bool
	exportCmdMeta       bool
	exportCmdStructure  bool
	exportCmdNoData     bool
	exportCmdTransform  bool
	exportCmdVis        bool
	exportCmdAll        bool
	exportCmdNameSpaced bool
	exportCmdZipped     bool
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "copy datasets to your local filesystem",
	Long: `
Export gets datasets out of qri. By default it exports only a dataset’s data to 
the path [current directory]/[peername]/[dataset name]/[data file]. 

To export everything about a dataset, use the --dataset flag.`,
	Annotations: map[string]string{
		"group": "dataset",
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		requireNotRPC(cmd.Name())
		path := cmd.Flag("output").Value.String()
		format := cmd.Flag("format").Value.String()
		dataFormat := cmd.Flag("data-format").Value.String()
		if dataFormat != "" && !(dataFormat == "json" || dataFormat == "csv" || dataFormat == "cbor") {
			ErrExit(fmt.Errorf("%s is not an accepted data format, options are json, csv, and cbor", dataFormat))
		}

		if blank, err := cmd.Flags().GetBool("blank"); err == nil && blank {
			if path == "" {
				path = "dataset.yaml"
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				err := ioutil.WriteFile(path, []byte(blankYamlDataset), os.ModePerm)
				ExitIfErr(err)
				printSuccess("blank dataset file saved to %s", path)
			} else {
				ErrExit(fmt.Errorf("'%s' already exists", path))
			}
			return
		}

		r := getRepo(false)
		req := core.NewDatasetRequests(r, nil)

		dsr, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		res := &repo.DatasetRef{}
		err = req.Get(&dsr, res)
		ExitIfErr(err)

		ds, err := res.DecodeDataset()
		ExitIfErr(err)

		if exportCmdNameSpaced {
			peerName := dsr.Peername
			if peerName == "me" {
				myProfile, err := r.Profile()
				if err != nil {
					ExitIfErr(err)
				}
				peerName = myProfile.Peername
			}
			path = filepath.Join(path, peerName)
		}
		path = filepath.Join(path, dsr.Name)

		if cmd.Flag("zip").Value.String() == "true" {
			dst, err := os.Create(fmt.Sprintf("%s.zip", path))
			ExitIfErr(err)

			err = dsutil.WriteZipArchive(r.Store(), ds, dst)
			ExitIfErr(err)
			err = dst.Close()
			ExitIfErr(err)
			return
		} else if exportCmdAll {
			exportCmdNoData = false
			exportCmdDataset = true
			exportCmdMeta = true
			exportCmdStructure = true
		}

		if path != "" {
			err = os.MkdirAll(path, os.ModePerm)
			ExitIfErr(err)
		}

		if exportCmdMeta {
			var md interface{}
			// TODO - this ensures a "form" metadata file is written
			// when one doesn't exist. This should be better
			if ds.Meta != nil && ds.Meta.IsEmpty() {
				md = struct {
					// Url to access the dataset
					AccessPath string `json:"accessPath"`
					// The frequency with which dataset changes. Must be an ISO 8601 repeating duration
					AccrualPeriodicity string `json:"accrualPeriodicity"`
					// Citations is a slice of assets used to build this dataset
					Citations []string `json:"citations"`
					// Contribute
					// Contributors []stirng `json:"contributors"`
					// Description follows the DCAT sense of the word, it should be around a paragraph of
					// human-readable text
					Description string `json:"description"`
					// Url that should / must lead directly to the data itself
					DownloadPath string `json:"downloadPath"`
					// HomePath is a path to a "home" resource, either a url or d.web path
					HomePath string `json:"homePath"`
					// Identifier is for *other* data catalog specifications. Identifier should not be used
					// or relied on to be unique, because this package does not enforce any of these rules.
					Identifier string `json:"identifier"`
					// String of Keywords
					Keywords []string `json:"keywords"`
					// Languages this dataset is written in
					Language []string `json:"language"`
					// License will automatically parse to & from a string value if provided as a raw string
					License string `json:"license"`
					// Kind is required, must be qri:md:[version]
					Qri dataset.Kind `json:"qri"`
					// path to readmePath
					ReadmePath string `json:"readmePath"`
					// Title of this dataset
					Title string `json:"title"`
					// Theme
					Theme []string `json:"theme"`
					// Version is the semantic version for this dataset
					Version string `json:"version"`
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
				ExitIfErr(err)
			default:
				mdBytes, err = yaml.Marshal(md)
				ExitIfErr(err)
				metaPath = fmt.Sprintf("%s.yaml", strings.TrimSuffix(metaPath, filepath.Ext(metaPath)))
			}
			err = ioutil.WriteFile(metaPath, mdBytes, os.ModePerm)
			ExitIfErr(err)
			printSuccess("exported metadata file to: %s", metaPath)
		}

		if exportCmdStructure {
			stPath := filepath.Join(path, dsfs.PackageFileStructure.Filename())
			var stBytes []byte

			switch format {
			case "json":
				stBytes, err = json.MarshalIndent(ds.Structure, "", "  ")
				ExitIfErr(err)
			default:
				stBytes, err = yaml.Marshal(ds.Structure)
				ExitIfErr(err)
				stPath = fmt.Sprintf("%s.yaml", strings.TrimSuffix(stPath, filepath.Ext(stPath)))
			}
			err = ioutil.WriteFile(stPath, stBytes, os.ModePerm)
			ExitIfErr(err)
			printSuccess("exported structure file to: %s", stPath)
		}

		if !exportCmdNoData {
			if dataFormat == "" {
				dataFormat = ds.Structure.Format.String()
			}

			df, err := dataset.ParseDataFormatString(dataFormat)
			ExitIfErr(err)

			p := &core.LookupParams{
				Format: df,
				Path:   ds.Path().String(),
				All:    true,
			}
			r := &core.LookupResult{}

			req, err := datasetRequests(true)
			ExitIfErr(err)

			err = req.LookupBody(p, r)
			ExitIfErr(err)

			dataPath := filepath.Join(path, fmt.Sprintf("data.%s", dataFormat))
			dst, err := os.Create(dataPath)
			ExitIfErr(err)

			_, err = dst.Write(r.Data)
			ExitIfErr(err)

			err = dst.Close()
			ExitIfErr(err)
			printSuccess("exported data to: %s", dataPath)
		}

		if exportCmdDataset {
			dsPath := filepath.Join(path, dsfs.PackageFileDataset.String())
			var dsBytes []byte

			switch format {
			case "json":
				dsBytes, err = json.MarshalIndent(ds, "", "  ")
				ExitIfErr(err)
			default:
				dsBytes, err = yaml.Marshal(ds)
				ExitIfErr(err)
				dsPath = fmt.Sprintf("%s.yaml", strings.TrimSuffix(dsPath, filepath.Ext(dsPath)))
			}
			err = ioutil.WriteFile(dsPath, dsBytes, os.ModePerm)
			ExitIfErr(err)

			printSuccess("exported dataset.json to: %s", dsPath)
		}

		// err = dsutil.WriteDir(r.Store(), ds, path)
		// ExitIfErr(err)
	},
}

func init() {
	RootCmd.AddCommand(exportCmd)
	exportCmd.Flags().BoolP("blank", "", false, "export a blank dataset YAML file, overrides all other flags except output")
	exportCmd.Flags().StringP("output", "o", "", "path to write to, default is current directory")
	exportCmd.Flags().StringP("format", "f", "yaml", "format for all exported files, except for data. yaml is the default format. options: yaml, json")
	exportCmd.Flags().StringP("data-format", "", "", "format for data file. default is the original data format. options: json, csv, cbor")
	exportCmd.Flags().BoolVarP(&exportCmdZipped, "zip", "z", false, "compress export as zip archive, export all parts of dataset, data in original format")
	exportCmd.Flags().BoolVarP(&exportCmdAll, "all", "a", false, "export full dataset package")
	exportCmd.Flags().BoolVarP(&exportCmdAll, "namespaced", "n", false, "export to a peer name namespaced directory")
	exportCmd.Flags().BoolVarP(&exportCmdDataset, "dataset", "", false, "export root dataset")
	exportCmd.Flags().BoolVarP(&exportCmdMeta, "meta", "m", false, "export dataset metadata file")
	exportCmd.Flags().BoolVarP(&exportCmdStructure, "structure", "s", false, "export dataset structure file")
	exportCmd.Flags().BoolVarP(&exportCmdNoData, "no-data", "", false, "don't include dataset data file in export")
	// exportCmd.Flags().BoolVarP(&exportCmdTransform, "transform", "t", false, "export dataset transform file")
	// exportCmd.Flags().BoolVarP(&exportCmdVis, "vis-conf", "c", false, "export viz config file")

	// TODO - get format conversion up & running
	// exportCmd.Flags().StringP("data-format", "", "csv", "set output format [csv,json,cbor]")
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
