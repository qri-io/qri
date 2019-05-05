package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	util "github.com/datatogether/api/apiutil"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
	"github.com/spf13/cobra"
)

// NewUpdateCommand creates a new `qri update` cobra command for updating datasets
func NewUpdateCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &UpdateOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "update",
		Short:   "schedule dataset updates",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
	}

	scheduleCmd := &cobra.Command{
		Use:   "schedule",
		Short: "schedule an update",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Schedule(args)
		},
	}
	unscheduleCmd := &cobra.Command{
		Use:   "unschedule",
		Short: "unschedule an update",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Unschedule(args)
		},
	}
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "list scheduled updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.List()
		},
	}

	listCmd.Flags().IntVar(&o.Page, "page", 1, "page number results, default 1")
	listCmd.Flags().IntVar(&o.PageSize, "page-size", 25, "page size of results, default 25")

	logsCmd := &cobra.Command{
		Use:     "logs",
		Aliases: []string{"log"},
		Short:   "show log of dataset updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Logs(args)
		},
	}

	logsCmd.Flags().IntVar(&o.Page, "page", 1, "page number results, default 1")
	logsCmd.Flags().IntVar(&o.PageSize, "page-size", 25, "page size of results, default 25")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "excute an update immideately",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.RunUpdate(args)
		},
	}

	runCmd.Flags().StringVarP(&o.Title, "title", "t", "", "title of commit message for update")
	runCmd.Flags().StringVarP(&o.Message, "message", "m", "", "commit message for update")
	runCmd.Flags().StringVarP(&o.Recall, "recall", "", "", "restore revisions from dataset history, only 'tf' applies when updating")
	runCmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	runCmd.Flags().BoolVarP(&o.Publish, "publish", "p", false, "publish successful update to the registry")
	runCmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "simulate updating a dataset")
	runCmd.Flags().BoolVarP(&o.NoRender, "no-render", "n", false, "don't store a rendered version of the the vizualization ")
	runCmd.Flags().StringSliceVarP(&o.FilePaths, "file", "f", nil, "dataset or component file (yaml or json)")
	runCmd.Flags().StringVarP(&o.BodyPath, "body", "", "", "path to file or url of data to add as dataset contents")
	runCmd.Flags().BoolVar(&o.Force, "force", false, "force a new commit, even if no changes are detected")
	runCmd.Flags().BoolVarP(&o.KeepFormat, "keep-format", "k", false, "convert incoming data to stored data format")

	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "control qri update daemon",
	}

	serviceStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "show update daemon status",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.ServiceStatus()
		},
	}
	serviceStartCmd := &cobra.Command{
		Use:   "start",
		Short: "start update daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			// warning: need to be very careful to *not* initialize an instance here
			// which is usually done by calling complete to trigger initialization
			// from the passed-in factory.
			return o.ServiceStart(ioStreams, f.QriRepoPath())
		},
	}
	serviceStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "stop update daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.ServiceStop()
		},
	}

	serviceCmd.AddCommand(serviceStartCmd, serviceStopCmd, serviceStatusCmd)

	cmd.AddCommand(
		scheduleCmd,
		unscheduleCmd,
		listCmd,
		logsCmd,
		runCmd,
		serviceCmd,
	)

	return cmd
}

// UpdateOptions encapsulates state for the update command
type UpdateOptions struct {
	ioes.IOStreams

	Ref     string
	Title   string
	Message string

	BodyPath  string
	FilePaths []string
	Recall    string

	Publish    bool
	DryRun     bool
	NoRender   bool
	Force      bool
	KeepFormat bool
	Secrets    []string

	Page     int
	PageSize int

	inst          *lib.Instance
	updateMethods *lib.UpdateMethods
}

// Complete adds any missing configuration that can only be added just before calling Run
func (o *UpdateOptions) Complete(f Factory, args []string) (err error) {
	if len(args) == 1 {
		o.Ref = args[0]
	}
	o.inst = f.Instance()
	o.updateMethods = lib.NewUpdateMethods(o.inst)
	return
}

// Schedule adds a job to the update scheduler
func (o *UpdateOptions) Schedule(args []string) (err error) {
	if len(args) < 1 {
		return lib.NewError(lib.ErrBadArgs, "please provide a dataset reference for updating")
	}
	p := &lib.ScheduleParams{
		Name:       args[0],
		SaveParams: o.saveParams(),
	}
	if len(args) > 1 {
		p.Periodicity = args[1]
	}

	res := &lib.Job{}
	if err := o.updateMethods.Schedule(p, res); err != nil {
		return err
	}

	printSuccess(o.IOStreams.ErrOut, "update scheduled, next update: %s\n", res.NextExec())
	return nil
}

// Unschedule removes a job from the scheduler
func (o *UpdateOptions) Unschedule(args []string) (err error) {
	if len(args) < 1 {
		return lib.NewError(lib.ErrBadArgs, "please provide a name to unschedule")
	}

	var (
		name = args[0]
		res  bool
	)
	if err := o.updateMethods.Unschedule(&name, &res); err != nil {
		return err
	}

	printSuccess(o.IOStreams.ErrOut, "unscheduled %s\n", args[0])
	return nil
}

// List shows scheduled update jobs
func (o *UpdateOptions) List() (err error) {
	// convert Page and PageSize to Limit and Offset
	page := util.NewPage(o.Page, o.PageSize)
	p := &lib.ListParams{
		Offset: page.Offset(),
		Limit:  page.Limit(),
	}
	res := []*lib.Job{}
	if err = o.updateMethods.List(p, &res); err != nil {
		return
	}

	for i, j := range res {
		num := p.Offset + i + 1
		printInfo(o.Out, "%d. %s\n  %s | %s\n", num, j.Name, j.Type, j.NextExec())
	}

	return
}

// Logs shows a history of job events
func (o *UpdateOptions) Logs(args []string) (err error) {
	if len(args) == 1 {
		return o.LogFile(args[0])
	}

	// convert Page and PageSize to Limit and Offset
	page := util.NewPage(o.Page, o.PageSize)
	p := &lib.ListParams{
		Offset: page.Offset(),
		Limit:  page.Limit(),
	}

	res := []*lib.Job{}
	if err = o.updateMethods.Logs(p, &res); err != nil {
		return
	}

	for i, j := range res {
		num := p.Offset + i + 1
		printInfo(o.Out, "%d. %s\n  %s | %s\n", num, j.Name, j.Type, j.NextExec())
	}

	return nil
}

// LogFile prints a log output file
func (o *UpdateOptions) LogFile(logName string) error {
	data := []byte{}
	if err := o.updateMethods.LogFile(&logName, &data); err != nil {
		return err
	}

	o.Out.Write(data)
	return nil
}

// ServiceStatus gets the current status of the update daemon
func (o *UpdateOptions) ServiceStatus() error {
	// TODO (b5):
	return fmt.Errorf("not finished")
}

// ServiceStart ensures the update service is running
func (o *UpdateOptions) ServiceStart(ioStreams ioes.IOStreams, repoPath string) (err error) {
	ctx := context.Background()

	cfgPath := filepath.Join(repoPath, "config.yaml")
	cfg, err := config.ReadFromFile(cfgPath)
	if err != nil {
		return err
	}
	if cfg.Update == nil {
		cfg.Update = config.DefaultUpdate()
	}

	qriPath, ipfsPath := EnvPathFactory()
	opts := []lib.Option{
		lib.OptLoadConfigFile(cfgPath),
		lib.OptIOStreams(ioStreams), // transfer iostreams down
		lib.OptSetQriRepoPath(qriPath),
		lib.OptSetIPFSPath(ipfsPath),
		lib.OptCheckConfigMigrations(""),
	}

	return lib.UpdateServiceStart(ctx, repoPath, cfg.Update, opts)
}

// ServiceStop halts the update scheduler service
func (o *UpdateOptions) ServiceStop() (err error) {
	// TODO (b5):
	return fmt.Errorf("not finished")
}

// RunUpdate executes an update immideately
func (o *UpdateOptions) RunUpdate(args []string) (err error) {
	if len(args) < 1 {
		return lib.NewError(lib.ErrBadArgs, "please provide the name of an update to run")
	}

	var (
		name = args[0]
		job  = &lib.Job{
			Name: name,
		}
	)

	o.StartSpinner()
	defer o.StopSpinner()

	res := &repo.DatasetRef{}
	if err := o.updateMethods.Run(job, res); err != nil {
		return err
	}

	printSuccess(o.Out, "updated dataset %s", res.AliasString())
	return nil
}

func (o *UpdateOptions) saveParams() *lib.SaveParams {
	p := &lib.SaveParams{
		Ref:                 o.Ref,
		Title:               o.Title,
		Message:             o.Message,
		BodyPath:            o.BodyPath,
		FilePaths:           o.FilePaths,
		Recall:              o.Recall,
		Publish:             o.Publish,
		DryRun:              o.DryRun,
		ShouldRender:        !o.NoRender,
		Force:               o.Force,
		ConvertFormatToPrev: o.KeepFormat,
	}

	if sec, err := parseSecrets(o.Secrets...); err != nil {
		log.Errorf("invalid secrets: %s", err.Error())
	} else {
		p.Secrets = sec
	}
	return p
}
