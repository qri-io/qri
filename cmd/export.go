package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "copy datasets to your local filesystem",
	Long: `
Usage:
	qri export [--dataset] [--meta] [--structure] [--data] <dataset ref…>

Export gets datasets out of qri. By default it exports only a dataset’s data to 
the path [current directory]/[peername]/[dataset name]/[data file]. 

To export everything about a dataset, use the --dataset flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("please specify a dataset name to export")
			return
		}
		path := cmd.Flag("output").Value.String()
		if path == "" {
			// TODO - support printing to stdout
			ErrExit(fmt.Errorf("please specify an output path"))
		}

		r := getRepo(false)
		req := core.NewDatasetRequests(r, nil)

		p := &core.GetDatasetParams{
			Name: args[0],
			Path: datastore.NewKey(args[0]),
		}
		res := &repo.DatasetRef{}
		err := req.Get(p, res)
		ExitIfErr(err)

		ds := res.Dataset

		if cmd.Flag("data-only").Value.String() == "true" {
			src, err := dsfs.LoadData(r.Store(), ds)
			ExitIfErr(err)

			dst, err := os.Create(fmt.Sprintf("%s.%s", path, ds.Structure.Format.String()))
			ExitIfErr(err)

			_, err = io.Copy(dst, src)
			ExitIfErr(err)

			err = dst.Close()
			ExitIfErr(err)
			return
		}

		if cmd.Flag("zip").Value.String() == "true" {
			dst, err := os.Create(fmt.Sprintf("%s.zip", path))
			ExitIfErr(err)

			err = dsutil.WriteZipArchive(r.Store(), ds, dst)
			ExitIfErr(err)
			err = dst.Close()
			ExitIfErr(err)
			return
		}

		err = dsutil.WriteDir(r.Store(), ds, path)
		ExitIfErr(err)
	},
}

func init() {
	RootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("output", "o", "dataset", "path to write to")
	exportCmd.Flags().BoolP("data-only", "d", false, "write data only (no package)")
	exportCmd.Flags().BoolP("zip", "z", false, "compress export as zip archive")
	// TODO - get format conversion up & running
	// exportCmd.Flags().StringP("format", "f", "csv", "set output format [csv,json]")
}
