package cmd

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	dataCmdFormat string
	dataCmdLimit  int
	dataCmdOffset int
	dataCmdAll    bool
)

// dataCmd represents the export command
var dataCmd = &cobra.Command{
	Use:   "data",
	Short: "read dataset data",
	Long: `
Data reads records from a dataset`,
	Example: `  show the first 50 rows of a dataset:
  $ qri data me/dataset_name`,
	Annotations: map[string]string{
		"group": "dataset",
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		loadConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("please specify a dataset name to retrieve data")
			return
		}

		r := getRepo(false)
		req := core.NewDatasetRequests(r, nil)

		dsr, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		res := &repo.DatasetRef{}
		err = req.Get(&dsr, res)
		ExitIfErr(err)

		ds := res.Dataset

		df, err := dataset.ParseDataFormatString(dataCmdFormat)
		ExitIfErr(err)

		p := &core.StructuredDataParams{
			Format: df,
			Path:   ds.Path().String(),
			Limit:  dataCmdLimit,
			Offset: dataCmdOffset,
			All:    dataCmdAll,
		}

		sd := &core.StructuredData{}

		if err := req.StructuredData(p, sd); err != nil {
			ErrExit(err)
		}

		data := sd.Data
		if p.Format == dataset.CBORDataFormat {
			data = []byte(hex.EncodeToString(sd.Data))
		}

		path := cmd.Flag("output").Value.String()
		if path != "" {
			ioutil.WriteFile(path, data, os.ModePerm)
		} else {
			fmt.Print(string(data))
			fmt.Println("")
		}
	},
}

func init() {
	RootCmd.AddCommand(dataCmd)
	dataCmd.Flags().StringP("output", "o", "", "path to write to, default is stdout")
	dataCmd.Flags().BoolVarP(&dataCmdAll, "all", "a", false, "read all dataset entries (overrides limit, offest)")
	dataCmd.Flags().StringVarP(&dataCmdFormat, "data-format", "f", "json", "format to export. one of [json,csv,cbor]")
	dataCmd.Flags().IntVarP(&dataCmdLimit, "limit", "l", 50, "max number of records to read")
	dataCmd.Flags().IntVarP(&dataCmdOffset, "offset", "s", 0, "number of records to skip")
}
