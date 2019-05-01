package cmd

import (
	"fmt"

	"github.com/qri-io/ioes"
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
		Use:   "list",
		Short: "list scheduled updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.List()
		},
	}
	logCmd := &cobra.Command{
		Use:   "log",
		Short: "show log of dataset updates",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.Log()
		},
	}

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
	// runCmd.Flags().BoolVarP(&o.Publish, "publish", "p", false, "publish this dataset to the registry")
	runCmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "simulate updating a dataset")
	runCmd.Flags().BoolVarP(&o.NoRender, "no-render", "n", false, "don't store a rendered version of the the vizualization ")

	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "control qri update daemon",
	}

	startServiceCmd := &cobra.Command{
		Use:   "start",
		Short: "start update daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.StartService()
		},
	}
	stopServiceCmd := &cobra.Command{
		Use:   "stop",
		Short: "stop update daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f, args); err != nil {
				return err
			}
			return o.StopService()
		},
	}

	serviceCmd.AddCommand(startServiceCmd, stopServiceCmd)

	cmd.AddCommand(
		scheduleCmd,
		unscheduleCmd,
		listCmd,
		logCmd,
		runCmd,
		serviceCmd,
	)

	return cmd
}

// UpdateOptions encapsulates state for the update command
type UpdateOptions struct {
	ioes.IOStreams

	Ref      string
	Title    string
	Message  string
	Recall   string
	Publish  bool
	DryRun   bool
	NoRender bool
	Secrets  []string

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
		Name: args[0],
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

	p := &lib.ListParams{
		// TODO (b5) - finish
		Limit:  100,
		Offset: 0,
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

// Log shows a history of job events
func (o *UpdateOptions) Log() (err error) {
	// TODO (b5):
	return fmt.Errorf("not finished")
}

// StartService ensures the update service is running
func (o *UpdateOptions) StartService() (err error) {
	var in, out bool
	return o.updateMethods.StartService(&in, &out)
}

// StopService halts the update scheduler service
func (o *UpdateOptions) StopService() (err error) {
	// TODO (b5):
	return fmt.Errorf("not finished")
}

// RunUpdate executes an update immideately
func (o *UpdateOptions) RunUpdate(args []string) (err error) {
	if len(args) < 1 {
		return lib.NewError(lib.ErrBadArgs, "please provide a name to unschedule")
	}

	// if o.Ref == "" {
	// 	return lib.NewError(lib.ErrBadArgs, "please provide a dataset reference for updating")
	// }
	// if o.Recall != "" && o.Recall != "tf" && o.Recall != "transform" {
	// 	return lib.NewError(lib.ErrBadArgs, "only 'tf' or 'transform' are valid recall values when updating")
	// }
	// return nil

	// 	p := &lib.UpdateParams{
	// 		Ref:          o.Ref,
	// 		Title:        o.Title,
	// 		Message:      o.Message,
	// 		DryRun:       o.DryRun,
	// 		Publish:      o.Publish,
	// 		ShouldRender: !o.NoRender,
	// 		ReturnBody:   false,
	// 	}

	// 	if o.Secrets != nil {
	// 		secretsMsg := `
	// Warning: You are providing secrets to a dataset transformation.
	// Never provide secrets to a transformation you do not trust.
	// continue?`
	// 		if !confirm(o.Out, o.In, secretsMsg, true) {
	// 			return
	// 		}

	// 		if p.Secrets, err = parseSecrets(o.Secrets...); err != nil {
	// 			return err
	// 		}
	// 	}
	var (
		name = args[0]
		job  = &lib.Job{}
	)

	if err = o.updateMethods.Job(&name, job); err != nil {
		// TODO (b5) - shouldn't require the job be scheduled to execute
		return err
	}

	o.StartSpinner()
	defer o.StopSpinner()

	res := &repo.DatasetRef{}
	if err := o.updateMethods.Run(job, res); err != nil {
		return err
	}

	printSuccess(o.Out, "updated dataset %s", res.AliasString())
	return nil
}
