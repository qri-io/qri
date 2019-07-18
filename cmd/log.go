package cmd

import (
	"fmt"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewLogCommand creates a new `qri log` cobra command
func NewLogCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &LogOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "log",
		Aliases: []string{"history"},
		Short:   "Show log of dataset history",
		Long: `
` + "`qri log`" + ` prints a list of changes to a dataset over time. Each entry in the log is a 
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

	return cmd
}

// LogOptions encapsulates state for the log command
type LogOptions struct {
	ioes.IOStreams

	PageSize int
	Page     int
	Refs     *RefSelect

	LogRequests *lib.LogRequests
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *LogOptions) Complete(f Factory, args []string) (err error) {
	if o.Refs, err = GetCurrentRefSelect(f, args, 1); err != nil {
		return err
	}
	o.LogRequests, err = f.LogRequests()
	return
}

// Run executes the log command
func (o *LogOptions) Run() error {

	// convert Page and PageSize to Limit and Offset
	page := util.NewPage(o.Page, o.PageSize)

	p := &lib.LogParams{
		Ref: o.Refs.Ref(),
		ListParams: lib.ListParams{
			Limit:  page.Limit(),
			Offset: page.Offset(),
		},
	}

	refs := []repo.DatasetRef{}
	if err := o.LogRequests.Log(p, &refs); err != nil {
		if err == repo.ErrEmptyRef {
			return lib.NewError(err, "please provide a dataset reference")
		}
		return err
	}

	items := make([]fmt.Stringer, len(refs))
	for i, r := range refs {
		items[i] = logStringer(r)
	}

	printItems(o.Out, items, page.Offset())
	return nil
}
