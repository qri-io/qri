package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewLogCommand creates a new `qri log` cobra command
func NewLogCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &LogOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "log [DATASET]",
		Aliases: []string{"history"},
		Short:   "show log of dataset commits",
		Long: "`qri log`" + ` lists dataset commits over time. Each entry in the log is a 
snapshot of a dataset taken at the moment it was saved that keeps exact details 
about how that dataset looked at at that point in time. 

We call these snapshots versions. Each version has an author (the peer that 
created the version) and a message explaining what changed. Log prints these 
details in order of occurrence, starting with the most recent known version, 
working backwards in time.

The log command can get the list of versions for a local dataset or a dataset
on the network at a remote.
`,
		Example: `  # Show log for the local dataset b5/precip:
  $ qri log b5/precip

  # Show log for a dataset on the Qri Cloud registry called ramfox/league_stats
  $ qri log ramfox/league_stats
	
  # Show log for a dataset chriswhong/nyc_parking_tickets on a remote named "nycdatacollection"
  $ qri log chriswhong/nyc_parking_tickets --source nycdatacollection`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Run()
		},
	}

	// cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json]")
	cmd.Flags().IntVar(&o.Offset, "offset", 0, "skip this number of records from the results, default 0")
	cmd.Flags().IntVar(&o.Limit, "limit", 25, "size of results, default 25")
	cmd.Flags().StringVarP(&o.Source, "source", "", "", "name of source to fetch from, disables local actions. `registry` will search the default qri registry")
	cmd.Flags().BoolVarP(&o.Local, "local", "l", false, "only fetch local logs, disables network actions")
	cmd.Flags().BoolVarP(&o.Pull, "pull", "p", false, "fetch the latest logs from the network")

	return cmd
}

// LogOptions encapsulates state for the log command
type LogOptions struct {
	ioes.IOStreams

	Offset int
	Limit  int
	Refs   *RefSelect
	Local  bool
	Pull   bool

	// remote fetching specific flags
	Source     string
	Unfetch    bool
	NoRegistry bool
	NoPin      bool

	Instance *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *LogOptions) Complete(f Factory, args []string) (err error) {
	if o.Instance, err = f.Instance(); err != nil {
		return err
	}

	if o.Local && (o.Source != "" || o.Pull) {
		return errors.New(err, "cannot use 'local' flag with either the 'source' or 'pull' flags")
	}

	if o.Refs, err = GetCurrentRefSelect(f, args, AnyNumberOfReferences); err != nil {
		if err == repo.ErrEmptyRef {
			return errors.New(err, "please provide a dataset reference")
		}
	}

	return err
}

// Run executes the log command
func (o *LogOptions) Run() error {

	ctx := context.TODO()
	p := &lib.ActivityParams{
		Ref:  o.Refs.Ref(),
		Pull: o.Pull,
		List: params.List{
			Offset: o.Offset,
			Limit:  o.Limit,
		},
	}

	res, err := o.Instance.WithSource(o.Source).Dataset().Activity(ctx, p)
	if err != nil {
		return err
	}

	makeItemsAndPrint(res, o.Out, o.Offset)
	return nil
}

func makeItemsAndPrint(refs []dsref.VersionInfo, out io.Writer, offset int) {
	items := make([]fmt.Stringer, len(refs))
	for i, r := range refs {
		items[i] = dslogItemStringer(r)
	}

	printItems(out, items, offset)
}

// NewLogbookCommand creates a `qri logbook` cobra command
func NewLogbookCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &LogbookOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:   "logbook [DATASET]",
		Short: "show a detailed list of changes on a dataset name",
		Example: `  # Show log for the dataset bob/precip:
  $ qri logbook bob/precip`,
		Long: `Logbooks are records of changes to a dataset. The logbook is more detailed
than a dataset history, recording the steps taken to construct a dataset history
without including dataset data. Logbooks can be synced with other users.

The logbook command shows entries for a dataset, from newest to oldest.`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			if o.Raw {
				return o.RawLogs()
			} else if o.Summary {
				return o.LogbookSummary()
			}
			return o.LogEntries()
		},
	}

	cmd.Flags().IntVar(&o.Offset, "offset", 0, "skip this number of records from the results, default 0")
	cmd.Flags().IntVar(&o.Limit, "limit", 25, "size of results, default 25")
	cmd.Flags().BoolVar(&o.Raw, "raw", false, "full logbook in raw JSON format. overrides all other flags")
	cmd.Flags().BoolVar(&o.Summary, "summary", false, "print one oplog per line in the format 'MODEL ID OPCOUNT NAME'. overrides all other flags")

	return cmd
}

// LogbookOptions encapsulates state for the log command
type LogbookOptions struct {
	ioes.IOStreams

	Offset       int
	Limit        int
	Refs         *RefSelect
	Raw, Summary bool

	Instance *lib.Instance
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *LogbookOptions) Complete(f Factory, args []string) (err error) {
	if o.Instance, err = f.Instance(); err != nil {
		return err
	}

	if o.Raw && o.Summary {
		return fmt.Errorf("cannot use summary & raw flags at once")
	}

	if o.Raw || o.Summary {
		if len(args) != 0 {
			return fmt.Errorf("can't use dataset reference. the raw flag shows the entire logbook")
		}
	} else {
		if o.Refs, err = GetCurrentRefSelect(f, args, 1); err != nil {
			return err
		}
	}

	return err
}

// LogEntries gets entries from the logbook
func (o *LogbookOptions) LogEntries() error {

	p := &lib.RefListParams{
		Ref:    o.Refs.Ref(),
		Offset: o.Offset,
		Limit:  o.Limit,
	}

	ctx := context.TODO()
	res, err := o.Instance.Log().Log(ctx, p)
	if err != nil {
		if err == repo.ErrEmptyRef {
			return errors.New(err, "please provide a dataset reference")
		}
		return err
	}

	// print items in reverse
	items := make([]fmt.Stringer, len(res))
	j := len(items)
	for _, r := range res {
		j--
		items[j] = logEntryStringer(r)
	}

	printItems(o.Out, items, o.Offset)
	return nil
}

// RawLogs executes the rawlogs variant of the logbook command
func (o *LogbookOptions) RawLogs() error {
	ctx := context.TODO()
	res, err := o.Instance.Log().RawLogbook(ctx, &lib.RawLogbookParams{})
	if err != nil {
		return err
	}

	data, err := json.Marshal(res)
	if err != nil {
		return err
	}

	printToPager(o.Out, bytes.NewBuffer(data))
	return nil
}

// LogbookSummary prints a logbook overview
func (o *LogbookOptions) LogbookSummary() error {
	ctx := context.TODO()
	res, err := o.Instance.Log().LogbookSummary(ctx, &struct{}{})
	if err != nil {
		return err
	}

	printToPager(o.Out, bytes.NewBufferString(*res))
	return nil
}
