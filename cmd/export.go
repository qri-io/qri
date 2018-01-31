package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	exportCmdDataset   bool
	exportCmdMeta      bool
	exportCmdStructure bool
	exportCmdData      bool
	exportCmdTransform bool
	exportCmdVis       bool
	exportCmdAll       bool
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "copy datasets to your local filesystem",
	Long: `
Export gets datasets out of qri. By default it exports only a dataset’s data to 
the path [current directory]/[peername]/[dataset name]/[data file]. 

To export everything about a dataset, use the --dataset flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("please specify a dataset name to export")
			return
		}
		path := cmd.Flag("output").Value.String()

		r := getRepo(false)
		req := core.NewDatasetRequests(r, nil)

		dsr, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		p := &repo.DatasetRef{
			Name: dsr.Name,
			Path: dsr.Path,
		}
		res := &repo.DatasetRef{}
		err = req.Get(p, res)
		ExitIfErr(err)

		ds := res.Dataset

		if cmd.Flag("zip").Value.String() == "true" {
			dst, err := os.Create(fmt.Sprintf("%s.zip", path))
			ExitIfErr(err)

			err = dsutil.WriteZipArchive(r.Store(), ds, dst)
			ExitIfErr(err)
			err = dst.Close()
			ExitIfErr(err)
			return
		} else if cmd.Flag("all").Value.String() == "true" {
			exportCmdData = true
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
			if ds.Meta.IsEmpty() {
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
			mdBytes, err := json.MarshalIndent(md, "", "  ")
			err = ioutil.WriteFile(metaPath, mdBytes, os.ModePerm)
			ExitIfErr(err)
			printSuccess("exported metadata file to: %s", metaPath)
		}

		if exportCmdStructure {
			stpath := filepath.Join(path, dsfs.PackageFileStructure.Filename())
			stbytes, err := json.MarshalIndent(ds.Structure, "", "  ")
			err = ioutil.WriteFile(stpath, stbytes, os.ModePerm)
			ExitIfErr(err)
			printSuccess("exported structure file to: %s", stpath)
		}

		if exportCmdData {
			src, err := dsfs.LoadData(r.Store(), ds)
			ExitIfErr(err)

			dataPath := filepath.Join(path, fmt.Sprintf("data.%s", ds.Structure.Format.String()))
			dst, err := os.Create(dataPath)
			ExitIfErr(err)

			_, err = io.Copy(dst, src)
			ExitIfErr(err)

			err = dst.Close()
			ExitIfErr(err)
			printSuccess("exported dataset data to: %s", dataPath)
		}

		if exportCmdDataset {
			dsPath := filepath.Join(path, dsfs.PackageFileDataset.String())
			dsbytes, err := json.MarshalIndent(ds, "", "  ")
			ExitIfErr(err)
			err = ioutil.WriteFile(dsPath, dsbytes, os.ModePerm)
			ExitIfErr(err)

			printSuccess("exported dataset dataset to: %s", dsPath)
		}

		// err = dsutil.WriteDir(r.Store(), ds, path)
		// ExitIfErr(err)
	},
}

func init() {
	RootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("output", "o", "", "path to write to, default is current directory")
	exportCmd.Flags().BoolP("zip", "z", false, "compress export as zip archive")
	exportCmd.Flags().BoolVarP(&exportCmdAll, "all", "a", false, "export full dataset package")
	exportCmd.Flags().BoolVarP(&exportCmdDataset, "dataset", "", false, "export root dataset")
	exportCmd.Flags().BoolVarP(&exportCmdMeta, "meta", "m", false, "export dataset metadata file")
	exportCmd.Flags().BoolVarP(&exportCmdStructure, "structure", "s", false, "export dataset structure file")
	exportCmd.Flags().BoolVarP(&exportCmdData, "data", "d", true, "export dataset data file")
	// exportCmd.Flags().BoolVarP(&exportCmdTransform, "transform", "t", false, "export dataset transform file")
	// exportCmd.Flags().BoolVarP(&exportCmdVis, "vis-conf", "c", false, "export viz config file")

	// TODO - get format conversion up & running
	// exportCmd.Flags().StringP("format", "f", "csv", "set output format [csv,json]")
}
