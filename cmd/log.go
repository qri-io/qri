package cmd

import (
	"fmt"

	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewLogCommand creates a new `qri log` cobra command
func NewLogCommand(f Factory, ioStreams IOStreams) *cobra.Command {
	o := &LogOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "log",
		Aliases: []string{"history"},
		Short:   "show log of dataset history",
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
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExitIfErr(o.ErrOut, o.Complete(f, args))
			ExitIfErr(o.ErrOut, o.Run())
		},
	}

	// cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json]")
	cmd.Flags().IntVarP(&o.Limit, "limit", "l", 25, "limit results, default 25")
	cmd.Flags().IntVarP(&o.Offset, "offset", "o", 0, "offset results, default 0")

	return cmd
}

// LogOptions encapsulates state for the log command
type LogOptions struct {
	IOStreams

	Limit  int
	Offset int
	Ref    string

	Repo            repo.Repo
	HistoryRequests *lib.HistoryRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *LogOptions) Complete(f Factory, args []string) (err error) {
	if f.RPC() != nil {
		return usingRPCError("log")
	}

	if len(args) < 1 {
		return fmt.Errorf("please specify a dataset reference to log")
	} else if len(args) != 1 {
		return fmt.Errorf("only one argument ([peername]/[datasetname]) allowed")
	}
	o.Ref = args[0]

	o.Repo, err = f.Repo()
	if err != nil {
		return err
	}
	o.HistoryRequests, err = f.HistoryRequests()
	return
}

// Run executes the log command
func (o *LogOptions) Run() error {

	ref, err := repo.ParseDatasetRef(o.Ref)
	if err != nil {
		return err
	}

	if err = repo.CanonicalizeDatasetRef(o.Repo, &ref); err != nil {
		return err
	}

	p := &lib.LogParams{
		Ref: ref,
		ListParams: lib.ListParams{
			Peername: ref.Peername,
			Limit:    o.Limit,
			Offset:   o.Offset,
		},
	}

	refs := []repo.DatasetRef{}
	if err = o.HistoryRequests.Log(p, &refs); err != nil {
		return err
	}

	for _, ref := range refs {
		printSuccess(o.Out, "%s - %s\n\t%s\n", ref.Dataset.Commit.Timestamp.Format("Jan _2 15:04:05"), ref.Path, ref.Dataset.Commit.Title)
	}

	// outformat := cmd.Flag("format").Value.String()
	// switch outformat {
	// case "":
	//  for _, ref := range refs {
	//    printInfo("%s\t\t\t: %s", ref.Name, ref.Path)
	//  }
	// case dataset.JSONDataFormat.String():
	//  data, err := json.MarshalIndent(refs, "", "  ")
	//  ExitIfErr(o.ErrOut, err)
	//  fmt.Printf("%s\n", string(data))
	// default:
	//  ErrExit(o.ErrOut, fmt.Errorf("unrecognized format: %s", outformat))
	// }
	return nil
}
