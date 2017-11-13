package cmd

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	sql "github.com/qri-io/dataset_sql"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	runCmdName string
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a query",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			ErrExit(fmt.Errorf("Please provide a query string to execute"))
		}

		r := GetRepo(false)
		req := core.NewQueryRequests(r)

		format, err := dataset.ParseDataFormatString(cmd.Flag("format").Value.String())
		if err != nil {
			ErrExit(fmt.Errorf("invalid data format: %s", cmd.Flag("format").Value.String()))
		}

		p := &core.RunParams{
			ExecOpt: sql.ExecOpt{
				Format: format,
			},
			SaveName: runCmdName,
			Dataset: &dataset.Dataset{
				Timestamp:   time.Now().In(time.UTC),
				QuerySyntax: "sql",
				QueryString: args[0],
			},
		}

		res := &repo.DatasetRef{}
		err = req.Run(p, res)
		ExitIfErr(err)

		f, err := dsfs.LoadData(r.Store(), res.Dataset)
		ExitIfErr(err)

		results, err := ioutil.ReadAll(f)
		ExitIfErr(err)

		PrintResults(res.Dataset.Structure, results, res.Dataset.Structure.Format)
	},
}

func init() {
	RootCmd.AddCommand(runCmd)
	// runCmd.Flags().StringP("save", "s", "", "save the resulting dataset to a given address")
	runCmd.Flags().StringP("output", "o", "", "file to write to")
	runCmd.Flags().StringP("format", "f", "csv", "set output format [csv,json]")
	runCmd.Flags().StringVarP(&runCmdName, "name", "n", "", "save output to name")
}
