package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/logbook"
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
  $ qri log chriswhong/nyc_parking_tickets --remote nycdatacollection`,
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
	cmd.Flags().IntVar(&o.PageSize, "page-size", 25, "page size of results, default 25")
	cmd.Flags().IntVar(&o.Page, "page", 1, "page number of results, default 1")
	cmd.Flags().StringVarP(&o.RemoteName, "remote", "", "", "name of remote to fetch from, disables local actions. `registry` will search the default qri registry")
	cmd.Flags().BoolVarP(&o.Local, "local", "l", false, "only fetch local logs, disables network actions")
	cmd.Flags().BoolVarP(&o.Pull, "pull", "p", false, "fetch the latest logs from the network")

	return cmd
}

// LogOptions encapsulates state for the log command
type LogOptions struct {
	ioes.IOStreams

	PageSize int
	Page     int
	Refs     *RefSelect
	Local    bool
	Pull     bool

	// remote fetching specific flags
	RemoteName string
	Unfetch    bool
	NoRegistry bool
	NoPin      bool

	LogMethods *lib.LogMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *LogOptions) Complete(f Factory, args []string) (err error) {
	if o.Local && (o.RemoteName != "" || o.Pull) {
		return errors.New(err, "cannot use 'local' flag with either the 'remote' or 'pull' flags")
	}

	if o.Refs, err = GetCurrentRefSelect(f, args, -1, nil); err != nil {
		if err == repo.ErrEmptyRef {
			return errors.New(err, "please provide a dataset reference")
		}
	}

	o.LogMethods, err = f.LogMethods()
	return
}

// DatasetLogItem aliases the type from logbook
type DatasetLogItem = logbook.DatasetLogItem

// Run executes the log command
func (o *LogOptions) Run() error {
	printRefSelect(o.ErrOut, o.Refs)

	// convert Page and PageSize to Limit and Offset
	page := util.NewPage(o.Page, o.PageSize)

	res := []DatasetLogItem{}
	p := &lib.LogParams{
		Ref:    o.Refs.Ref(),
		Pull:   o.Pull,
		Source: o.RemoteName,
		ListParams: lib.ListParams{
			Limit:  page.Limit(),
			Offset: page.Offset(),
		},
	}

	if err := o.LogMethods.Log(p, &res); err != nil {
		return err
	}

	makeItemsAndPrint(res, o.Out, page)
	return nil
}

func makeItemsAndPrint(refs []DatasetLogItem, out io.Writer, page util.Page) {
	items := make([]fmt.Stringer, len(refs))
	for i, r := range refs {
		items[i] = dslogItemStringer(r)
	}

	printItems(out, items, page.Offset())
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
			}
			return o.Logbook()
		},
	}

	// cmd.Flags().StringVarP(&o.Format, "format", "f", "", "set output format [json]")
	cmd.Flags().IntVar(&o.PageSize, "page-size", 25, "page size of results, default 25")
	cmd.Flags().IntVar(&o.Page, "page", 1, "page number of results, default 1")
	cmd.Flags().BoolVar(&o.Raw, "raw", false, "full logbook in raw JSON format. overrides all other flags")

	return cmd
}

// LogbookOptions encapsulates state for the log command
type LogbookOptions struct {
	ioes.IOStreams

	PageSize int
	Page     int
	Refs     *RefSelect
	Raw      bool

	LogMethods *lib.LogMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *LogbookOptions) Complete(f Factory, args []string) (err error) {
	if o.Raw {
		if len(args) != 0 {
			return fmt.Errorf("can't use dataset reference. the raw flag shows the entire logbook")
		}
	} else {
		if o.Refs, err = GetCurrentRefSelect(f, args, 1, nil); err != nil {
			return err
		}
	}

	o.LogMethods, err = f.LogMethods()
	return
}

// Logbook executes the Logbook command
func (o *LogbookOptions) Logbook() error {
	printRefSelect(o.ErrOut, o.Refs)

	// convert Page and PageSize to Limit and Offset
	page := util.NewPage(o.Page, o.PageSize)

	p := &lib.RefListParams{
		Ref:    o.Refs.Ref(),
		Limit:  page.Limit(),
		Offset: page.Offset(),
	}

	res := []lib.LogEntry{}
	if err := o.LogMethods.Logbook(p, &res); err != nil {
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

	printItems(o.Out, items, page.Offset())
	return nil
}

// RawLogs executes the rawlogs variant of the logbook command
func (o *LogbookOptions) RawLogs() error {
	res := lib.PlainLogs{}
	if err := o.LogMethods.PlainLogs(&lib.PlainLogsParams{}, &res); err != nil {
		return err
	}

	data, err := json.Marshal(res)
	if err != nil {
		return err
	}

	printToPager(o.Out, bytes.NewBuffer(data))
	return nil
}
