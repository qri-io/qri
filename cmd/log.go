package cmd

import (
	"fmt"
	// "encoding/json"
	// "fmt"
	// "github.com/qri-io/dataset"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

var (
	dsLogLimit, dsLogOffset int
	dsLogName               string
)

var datasetLogCmd = &cobra.Command{
	Use: "log",
	// Aliases: []string{"ls"},
	Short: "show log of dataset history",
	Long: `
log prints a list of changes to a dataset over time. Each entry in the log is a 
snapshot of a dataset taken at the moment it was saved that keeps exact details 
about how that dataset looked at at that point in time. 

We call these snapshots versions. Each version has an author (the peer that 
created the version) and a message explaining what changed. Log prints these 
details in order of occurrence, starting with the most recent known version, 
working backwards in time.`,
	Example: `  show log for the dataset b5/precip:
	$ qri log b5/precip`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			ErrExit(fmt.Errorf("please specify a dataset reference to log"))
		} else if len(args) != 1 {
			ErrExit(fmt.Errorf("only one argument ([peername]/[datasetname]) allowed"))
		}

		online := false

		r := getRepo(false)
		ref, err := repo.ParseDatasetRef(args[0])
		ExitIfErr(err)

		err = repo.CanonicalizeDatasetRef(r, &ref)
		ExitIfErr(err)

		if ref.Path == "" {
			online = true
		}

		// TODO - add limit & offset params
		hr, err := historyRequests(online)
		ExitIfErr(err)

		p := &core.LogParams{
			// Limit:  dsLogLimit,
			// Offset: dsLogOffset,
			Ref: ref,
			ListParams: core.ListParams{
				Peername: ref.Peername,
			},
		}

		refs := []repo.DatasetRef{}
		err = hr.Log(p, &refs)
		ExitIfErr(err)

		for _, ref := range refs {
			printSuccess("%s - %s\n\t%s\n", ref.Dataset.Commit.Timestamp.Format("Jan _2 15:04:05"), ref.Path, ref.Dataset.Commit.Title)
		}

		// outformat := cmd.Flag("format").Value.String()
		// switch outformat {
		// case "":
		// 	for _, ref := range refs {
		// 		printInfo("%s\t\t\t: %s", ref.Name, ref.Path)
		// 	}
		// case dataset.JSONDataFormat.String():
		// 	data, err := json.MarshalIndent(refs, "", "  ")
		// 	ExitIfErr(err)
		// 	fmt.Printf("%s\n", string(data))
		// default:
		// 	ErrExit(fmt.Errorf("unrecognized format: %s", outformat))
		// }

	},
}

func init() {
	RootCmd.AddCommand(datasetLogCmd)
	// datasetLogCmd.Flags().StringP("format", "f", "", "set output format [json]")
	datasetLogCmd.Flags().IntVarP(&dsLogLimit, "limit", "l", 25, "limit results, default 25")
	datasetLogCmd.Flags().IntVarP(&dsLogOffset, "offset", "o", 0, "offset results, default 0")
	datasetLogCmd.Flags().StringVarP(&dsLogName, "name", "n", "", "name of dataset to get logs for")
}
