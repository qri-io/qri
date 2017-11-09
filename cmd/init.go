package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	initFile     string
	initMetaFile string
	initName     string
	initUrl      string
	initPassive  bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a dataset, adding it to your local collection of datasets",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var dataFile, metaFile *os.File

		if initFile == "" && initUrl == "" {
			ErrExit(fmt.Errorf("please provide either a file or a url argument"))
		} else if initName == "" {
			ErrExit(fmt.Errorf("please provide a --name"))
		}

		err := loadFileIfPath(initFile, dataFile)
		ExitIfErr(err)
		err = loadFileIfPath(initMetaFile, metaFile)
		ExitIfErr(err)

		r := GetRepo(false)
		store := GetIpfsFilestore(false)
		req := core.NewDatasetRequests(store, r)

		p := &core.InitDatasetParams{
			Name:         initName,
			Url:          initUrl,
			DataFilename: filepath.Base(initFile),
		}

		// this is because passing nil to interfaces is bad
		// see: https://golang.org/doc/faq#nil_error
		if dataFile != nil {
			p.Data = dataFile
		}
		if metaFile != nil {
			p.Metadata = metaFile
		}

		ref := &repo.DatasetRef{}
		err = req.InitDataset(p, ref)
		ExitIfErr(err)
		// req.Get(&core.GetDatasetParams{ Name: p.Name }, res)
		PrintSuccess("initialized dataset %s: %s", ref.Name, ref.Path.String())
	},
}

func init() {
	flag.Parse()
	RootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initUrl, "url", "u", "", "url to file to initialize from")
	initCmd.Flags().StringVarP(&initFile, "file", "f", "", "data file to initialize from")
	initCmd.Flags().StringVarP(&initName, "name", "n", "", "name to give dataset")
	initCmd.Flags().StringVarP(&initMetaFile, "meta", "m", "", "dataset metadata")
	initCmd.Flags().BoolVarP(&initPassive, "passive", "p", false, "disable interactive init")
}
