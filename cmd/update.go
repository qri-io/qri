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
		Use:   "update",
		Short: "schedule dataset updates",
		Long: `The qri update commands allow you to schedule automatic,
periodic udates to your dataset. It also allows you to start & stop the
updating daemon, to view the log of past updates and list of upcoming
updates.
	`,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
	}

	scheduleCmd := &cobra.Command{
		Use:   "schedule",
		Short: "schedule an update",
		Long: `Schedule a dataset using all the same flags as the save command,
except you must provide a periodicity (or have a periodicity set in your 
dataset's Meta component. The given periodicity must be in the ISO 8601
repeated duration format (for more information of ISO 8601, check out 
	https://www.digi.com/resources/documentation/digidocs/90001437-13/reference/r_iso_8601_duration_format.htm)

Like the update Run command, the Schedule command assumes you want to recall 
the most recent transform in the dataset.

You can schedule an update for a dataset that has already been created, 
a dataset you are creating for the first time, or a shell script that 
calls "qri save" to update a dataset.

IMPORTANT: use the "qri update service" status command to ensure that the process
responsible for executing your scheduled updates is currently active.
	`,
		Example: `  schedule the weekly update of a dataset you have already created
	$ qri update schedule b5/my_dataset R/P1W
	qri scheduled b5/my_dataset, next update: 2019-05-14 20:15:13.191602 +0000 UTC
	
	schedule the daily update of a dataset that you are creating for the first 
	time:
	$ qri update schedule --file dataset.yaml b5/my_dataset R/P1D
	qri scheduled b5/my_dataset, next update: 2019-05-08 20:15:13.191602 +0000 UTC
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Schedule(args)
		},
	}

	scheduleCmd.Flags().StringVarP(&o.Title, "title", "t", "", "title of commit message for update")
	scheduleCmd.Flags().StringVarP(&o.Message, "message", "m", "", "commit message for update")
	scheduleCmd.Flags().StringVarP(&o.Recall, "recall", "", "", "restore revisions from dataset history, only 'tf' applies when updating")
	scheduleCmd.Flags().StringSliceVar(&o.Secrets, "secrets", nil, "transform secrets as comma separated key,value,key,value,... sequence")
	scheduleCmd.Flags().BoolVarP(&o.Publish, "publish", "p", false, "publish successful update to the registry")
	scheduleCmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "simulate updating a dataset")
	scheduleCmd.Flags().BoolVarP(&o.NoRender, "no-render", "n", false, "don't store a rendered version of the the vizualization ")
	scheduleCmd.Flags().StringSliceVarP(&o.FilePaths, "file", "f", nil, "dataset or component file (yaml or json)")
	scheduleCmd.Flags().StringVarP(&o.BodyPath, "body", "", "", "path to file or url of data to add as dataset contents")
	scheduleCmd.Flags().BoolVar(&o.Force, "force", false, "force a new commit, even if no changes are detected")
	scheduleCmd.Flags().BoolVarP(&o.KeepFormat, "keep-format", "k", false, "convert incoming data to stored data format")
	scheduleCmd.Flags().StringVar(&o.RepoPath, "use-repo", "", "experiment. run update on behalf of another repo")

	unscheduleCmd := &cobra.Command{
		Use:   "unschedule",
		Short: "unschedule an update",
		Long: `Unscheduling an update removes that dataset from the list of
scheduled updates.
	`,
		Example: `  unschedule an update using the dataset name
	$ qri update unschedule b5/my_dataset
	unscheduled b5/my_dataset
	
	`,
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
		Long: `Update list gives you a view into the upcoming scheduled updates, starting
with the most immediate update.
	`,
		Example: `  list the upcoming updates:
  $ qri update list
  1. b5/my_dataset
  dataset | 2019-05-08 16:19:23 -0400 EDT

  2. b5/my_next_dataset
  dataset | 2019-05-09 16:19:23 -0400 EDT

	`,
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
		Long: `Update logs shows the log of the updates that have already run,
starting with the most recent. The log includes a timestamped name, the type of
update that occured (dataset or shell), and the time it occured.

Using the name of a specific log as a parameter gives you the output of that
update.
	`,
		Example: `  list the log of previous updates:
  $ qri update logs
  1. 1557173933-my_dataset
  dataset | 2019-05-06 16:19:23 -0400 EDT

  2. 1557173585-my_dataset
  dataset | 2019-05-06 16:13:35 -0400 EDT
  ...

  get the output of one specific update:
  $ qri update log 1557173933-my_dataset
  dataset saved: b5/my_dataset@MSN9/ipfs/BntM
	`,
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
		Short: "execute an update immediately",
		Long: `Run allows you to execute an update immediately, rather then wait for
its scheduled time. Run uses the same parameters as the Save command, but
but assumes you want to recall the most recent transform in the dataset.
	`,
		Example: `  run an update
  $ qri update run b5/my_dataset
  🤖  running transform...
  ✅ transform complete
  dataset saved: b5/my_dataset@MSN9/ipfs/2BntM

	`,
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
		Long: `The qri service commands allow you to start, stop, and check the 
status of update daemon that executes the dataset updates.`,
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
		Long: `Use the "qri update service start" command to begin the process 
responsible for executing your scheduled updates. Any updates that were
scheduled before the update daemon has started will be run as soon as the 
updating process initiates.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// warning: need to be very careful to *not* initialize an instance here
			// which is usually done by calling complete to trigger initialization
			// from the passed-in factory.
			return o.ServiceStart(ioStreams, f.QriRepoPath())
		},
	}

	serviceStartCmd.Flags().BoolVarP(&o.Daemonize, "daemonize", "d", false, "start service as as a long-lived daemon")

	serviceStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "stop update daemon",
		Long: `Use the "qri update service stop" command to end the process 
responsible for executing your scheduled updates. With this process terminated,
your scheduled updates will not run.`,
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

	Daemonize bool
	Page      int
	PageSize  int

	// specifies custom repo location when scheduling a job,
	// should only be set if --repo persistent flag is set
	RepoPath string

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
		RepoPath:   o.RepoPath,
	}
	if len(args) > 1 {
		p.Periodicity = args[1]
	}

	res := &lib.Job{}
	if err := o.updateMethods.Schedule(p, res); err != nil {
		return err
	}

	printSuccess(o.ErrOut, "update scheduled, next update: %s\n", res.NextExec())
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

	printSuccess(o.ErrOut, "update unscheduled: %s\n", args[0])
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

	items := make([]fmt.Stringer, len(res))
	for i, r := range res {
		items[i] = jobStringer(*r)
	}
	printItems(o.Out, items)
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

	items := make([]fmt.Stringer, len(res))
	for i, r := range res {
		items[i] = jobStringer(*r)
	}
	printItems(o.Out, items)
	return
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
	var in bool
	res := &lib.ServiceStatus{}
	if err := o.updateMethods.ServiceStatus(&in, res); err != nil {
		return err
	}

	fmt.Fprint(o.Out, res.Name)
	return nil
}

// ServiceStart ensures the update service is running
func (o *UpdateOptions) ServiceStart(ioStreams ioes.IOStreams, repoPath string) (err error) {
	var res bool
	ctx := context.Background()

	cfgPath := filepath.Join(repoPath, "config.yaml")
	cfg, err := config.ReadFromFile(cfgPath)
	if err != nil {
		return err
	}
	if cfg.Update == nil {
		cfg.Update = config.DefaultUpdate()
	}

	p := &lib.UpdateServiceStartParams{
		Ctx:       ctx,
		Daemonize: o.Daemonize,

		UpdateCfg: cfg.Update,
		RepoPath:  repoPath,
	}
	return o.updateMethods.ServiceStart(p, &res)
}

// ServiceStop halts the update scheduler service
func (o *UpdateOptions) ServiceStop() (err error) {
	var in, out bool
	return o.updateMethods.ServiceStop(&in, &out)
}

// RunUpdate executes an update immediately
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
