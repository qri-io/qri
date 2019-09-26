package lib

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/update"
	"github.com/qri-io/qri/update/cron"
)

// NewUpdateMethods creates a configuration handle from an instance
func NewUpdateMethods(inst *Instance) *UpdateMethods {
	m := &UpdateMethods{
		inst: inst,
	}

	return m
}

// UpdateMethods enapsulates logic for scheduled updates
type UpdateMethods struct {
	inst *Instance
}

// CoreRequestsName specifies this is a Methods object
func (m *UpdateMethods) CoreRequestsName() string {
	return "update"
}

// Job aliases a cron.Job, removing the need to import the cron package to work
// with lib.UpdateMethods
type Job = cron.Job

// ScheduleParams encapsulates parameters for scheduling updates
type ScheduleParams struct {
	Name        string
	Periodicity string
	RepoPath    string

	// SaveParams only applies to dataset saves
	SaveParams *SaveParams
}

// Schedule creates a job and adds it to the scheduler
func (m *UpdateMethods) Schedule(in *ScheduleParams, out *cron.Job) (err error) {
	// this context is scoped to the scheduling request. currently not cancellable
	// because our lib methods don't accept a context themselves
	// TODO (b5): refactor RPC communication to use context
	ctx := context.TODO()

	// Make all paths absolute. this must happen *before* any possible RPC call
	if update.PossibleShellScript(in.Name) {
		if err = qfs.AbsPath(&in.Name); err != nil {
			return
		}
	}

	if err = in.SaveParams.AbsolutizePaths(); err != nil {
		return err
	}

	if m.inst.rpc != nil {
		return fmt.Errorf("apologies, we currently can't schedule updates while `qri connect` is running")
		// TODO (b5) - this returns error EOF when run. need to figure out why. I suspect it's b/c
		// job contains an interface field: Options
		// return m.inst.rpc.Call("UpdateMethods.Schedule", in, out)
	}

	job, err := m.jobFromScheduleParams(ctx, in)
	if err != nil {
		return err
	}

	if m.inst.cron == nil {
		return fmt.Errorf("update service not available")
	}

	err = m.inst.cron.Schedule(ctx, job)
	*out = *job
	return err
}

func (m *UpdateMethods) jobFromScheduleParams(ctx context.Context, p *ScheduleParams) (job *cron.Job, err error) {
	if update.PossibleShellScript(p.Name) {
		return update.ShellScriptToJob(p.Name, p.Periodicity, nil)
	}

	var ref repo.DatasetRef
	if ref, err = repo.ParseDatasetRef(p.Name); err != nil {
		return
	}

	r := m.inst.Repo()

	// if the job specifies a foreign repo path, create a temporary instance to read foreign dataset.
	// TODO (b5) - this currently should require the repo in question not be used via a file lock
	// it should then be upgraded to operate over RPC if the lock is held by a process
	if p.RepoPath != "" && p.RepoPath != m.inst.RepoPath() {
		var finst *Instance
		ctx, done := context.WithCancel(m.inst.Context())
		defer done()

		if finst, err = NewInstance(ctx, p.RepoPath); err != nil {
			return
		}
		r = finst.Repo()
	}

	if err = repo.CanonicalizeDatasetRef(r, &ref); err != nil {
		return
	}
	if err = base.ReadDataset(ctx, r, &ref); err != nil {
		return
	}

	var o *cron.DatasetOptions
	if p.SaveParams != nil {
		o = &cron.DatasetOptions{
			Title:     p.SaveParams.Title,
			Message:   p.SaveParams.Message,
			Recall:    p.SaveParams.Recall,
			BodyPath:  p.SaveParams.BodyPath,
			FilePaths: p.SaveParams.FilePaths,
			Publish:   p.SaveParams.Publish,
			// Strict:              p.SaveParams.Strict,
			Force:               p.SaveParams.Force,
			ConvertFormatToPrev: p.SaveParams.ConvertFormatToPrev,
			ShouldRender:        p.SaveParams.ShouldRender,
			Secrets:             p.SaveParams.Secrets,
			// TODO (b5) not fully supported yet:
			// Config: p.SaveParams.
		}
	}

	return update.DatasetToJob(ref.Dataset, p.Periodicity, o)
}

// Unschedule removes a job from the scheduler by name
func (m *UpdateMethods) Unschedule(name *string, unscheduled *bool) error {
	// this context is scoped to the scheduling request. currently not cancellable
	// because our lib methods don't accept a context themselves
	// TODO (b5): refactor RPC communication to use context
	var ctx = context.Background()

	return m.inst.cron.Unschedule(ctx, *name)
}

// List gets scheduled jobs
func (m *UpdateMethods) List(p *ListParams, jobs *[]*Job) error {
	// this context is scoped to the scheduling request. currently not cancellable
	// because our lib methods don't accept a context themselves
	// TODO (b5): refactor RPC communication to use context
	var ctx = context.Background()

	list, err := m.inst.cron.ListJobs(ctx, p.Offset, p.Limit)
	if err != nil {
		return err
	}

	*jobs = list
	return nil
}

// Job gets a job by name
func (m *UpdateMethods) Job(name *string, job *Job) (err error) {
	// this context is scoped to the scheduling request. currently not cancellable
	// because our lib methods don't accept a context themselves
	// TODO (b5): refactor RPC communication to use context
	var ctx = context.Background()

	res, err := m.inst.cron.Job(ctx, *name)
	if err != nil {
		return err
	}

	*job = *res
	return nil
}

// Logs shows the history of job execution
func (m *UpdateMethods) Logs(p *ListParams, res *[]*Job) error {
	// this context is scoped to the scheduling request. currently not cancellable
	// because our lib methods don't accept a context themselves
	// TODO (b5): refactor RPC communication to use context
	var ctx = context.Background()

	jobs, err := m.inst.cron.ListLogs(ctx, p.Offset, p.Limit)
	if err != nil {
		return err
	}

	*res = jobs
	return nil
}

// LogFile reads log file data for a given logName
func (m *UpdateMethods) LogFile(logName *string, data *[]byte) error {
	f, err := m.inst.cron.LogFile(context.Background(), *logName)
	if err != nil {
		return err
	}

	defer f.Close()
	res, err := ioutil.ReadAll(f)
	*data = res

	return err
}

// ServiceStatus describes the current state of a service
type ServiceStatus struct {
	Name       string
	Running    bool
	Daemonized bool // if true this service is scheduled
	Started    *time.Time
	Address    string
	Metrics    map[string]interface{}
}

// ServiceStatus reports status of the update daemon
func (m *UpdateMethods) ServiceStatus(in *bool, out *ServiceStatus) error {
	res, err := update.Status()
	if err != nil {
		return err
	}

	*out = ServiceStatus{
		Name: res,
	}
	return nil
}

// UpdateServiceStartParams configures startup
type UpdateServiceStartParams struct {
	Ctx       context.Context
	Daemonize bool

	// TODO (b5): I'm really not a fan of passing these configuration-derived
	// bits as parameters. Ideally this would come from the underlying instance
	// these are needed because lib.NewInstance creates a cron client
	// that intereferes with the start service process. We're currently getting
	// around this by avoiding calls to lib.NewInstance, or passing in resulting
	// params when called. We should clean this up.
	RepoPath  string
	UpdateCfg *config.Update
}

// ServiceStart ensures the scheduler is running
func (m *UpdateMethods) ServiceStart(p *UpdateServiceStartParams, started *bool) error {
	// TODO (b5) - these work when the API is running
	if p.RepoPath == "" && m.inst != nil {
		p.RepoPath = m.inst.RepoPath()
	}
	if p.UpdateCfg == nil && m.inst != nil {
		p.UpdateCfg = m.inst.Config().Update
	}

	if !p.Daemonize {
		log.Info("starting update service")
	}

	*started = true
	return update.Start(p.Ctx, p.RepoPath, p.UpdateCfg, p.Daemonize)
}

// ServiceStop halts the scheduler
func (m *UpdateMethods) ServiceStop(in, out *bool) error {
	*out = true
	return update.StopDaemon(m.inst.RepoPath())
}

// ServiceRestart uses shell commands to restart the scheduler service
func (m *UpdateMethods) ServiceRestart(in, out *bool) error {
	// TODO (b5):
	return fmt.Errorf("not finished")
}

// Run advances a dataset to the latest known version from either a peer or by
// re-running a transform in the peer's namespace
func (m *UpdateMethods) Run(p *Job, res *repo.DatasetRef) (err error) {
	// Make all paths absolute. this must happen *before* any possible RPC call
	if update.PossibleShellScript(p.Name) {
		if err = qfs.AbsPath(&p.Name); err != nil {
			return
		}
		p.Type = cron.JTShellScript
	} else {
		p.Type = cron.JTDataset
	}

	if err := absolutizeJobFilepaths(p); err != nil {
		return err
	}

	if m.inst.rpc != nil {
		return m.inst.rpc.Call("UpdateMethods.Run", p, res)
	}
	ctx := context.TODO()

	switch p.Type {
	case cron.JTDataset:
		params := &SaveParams{
			Ref: p.Name,
		}
		if o, ok := p.Options.(*cron.DatasetOptions); ok {
			params = &SaveParams{
				Ref:                 p.Name,
				Title:               o.Title,
				Message:             o.Message,
				Recall:              o.Recall,
				BodyPath:            o.BodyPath,
				FilePaths:           o.FilePaths,
				Publish:             o.Publish,
				Force:               o.Force,
				ConvertFormatToPrev: o.ConvertFormatToPrev,
				ShouldRender:        o.ShouldRender,
				Secrets:             o.Secrets,

				// TODO (b5) not fully supported yet:
				// Strict: o.Strict,
				// Config: o.Config
			}
		}
		*res = repo.DatasetRef{}
		err = m.runDatasetUpdate(ctx, params, res)

	case cron.JTShellScript:
		return update.JobToCmd(m.inst.streams, p).Run()

	case cron.JobType(""):
		return fmt.Errorf("update requires a job type to run")
	default:
		return fmt.Errorf("unrecognized update type: %s", p.Type)
	}

	if err != nil {
		return err
	}

	// TODO (b5): expand event logging interface to support storing additional details
	// return m.inst.Repo().LogEvent(repo.ETCronJobRan, *res)
	return nil
}

func absolutizeJobFilepaths(j *Job) error {
	if o, ok := j.Options.(*cron.DatasetOptions); ok {
		if err := qfs.AbsPath(&o.BodyPath); err != nil {
			return err
		}
		for i := range o.FilePaths {
			if err := qfs.AbsPath(&o.FilePaths[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *UpdateMethods) runDatasetUpdate(ctx context.Context, p *SaveParams, res *repo.DatasetRef) error {
	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}

	if err = repo.CanonicalizeDatasetRef(m.inst.node.Repo, &ref); err == repo.ErrNotFound {
		return fmt.Errorf("unknown dataset '%s'. please add before updating", ref.AliasString())
	} else if err != nil {
		return err
	}

	if !base.InLocalNamespace(m.inst.Repo(), &ref) {
		*res, err = actions.UpdateRemoteDataset(ctx, m.inst.Node(), &ref, true)
		return err
	}

	// default to recalling transform scripts for local updates
	// TODO (b5): not sure if this should be here or in client libraries
	if p.Recall == "" {
		p.Recall = "tf"
	}

	dsr := NewDatasetRequestsInstance(m.inst)
	return dsr.Save(p, res)
}
