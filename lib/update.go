package lib

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/cron"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/update"
)

// NewUpdateMethods creates a configuration handle from an instance
func NewUpdateMethods(inst *Instance) *UpdateMethods {
	m := &UpdateMethods{
		inst:        inst,
		scriptsPath: filepath.Join(inst.QriPath(), "update_scripts"),
	}

	if err := os.MkdirAll(m.scriptsPath, os.ModePerm); err != nil {
		log.Errorf("creating update scripts directory: %s", err.Error())
	}

	return m
}

// UpdateMethods enapsulates logic for scheduled updates
type UpdateMethods struct {
	inst        *Instance
	scriptsPath string
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

	// SaveParams only applies to dataset saves
	SaveParams *SaveParams
}

// Schedule creates a job and adds it to the scheduler
func (m *UpdateMethods) Schedule(in *ScheduleParams, out *cron.Job) (err error) {

	// Make all paths absolute. this must happen *before* any possible RPC call
	if base.PossibleShellScript(in.Name) {
		if err = qfs.AbsPath(&in.Name); err != nil {
			return
		}
	}

	if in.SaveParams != nil {
		for i := range in.SaveParams.FilePaths {
			if err := qfs.AbsPath(&in.SaveParams.FilePaths[i]); err != nil {
				return err
			}
		}

		if err := qfs.AbsPath(&in.SaveParams.BodyPath); err != nil {
			return fmt.Errorf("body file: %s", err)
		}
	}

	if m.inst.rpc != nil {
		return m.inst.rpc.Call("UpdateMethods.Schedule", in, out)
	}

	job, err := m.jobFromScheduleParams(in)
	if err != nil {
		return err
	}
	*out = *job

	// this context is scoped to the scheduling request. currently not cancellable
	// because our lib methods don't accept a context themselves
	// TODO (b5): refactor RPC communication to use context
	var ctx = context.Background()

	if m.inst.cron == nil {
		return fmt.Errorf("update service not available")
	}

	return m.inst.cron.Schedule(ctx, job)
}

func (m *UpdateMethods) jobFromScheduleParams(p *ScheduleParams) (job *cron.Job, err error) {
	if base.PossibleShellScript(p.Name) {
		// TODO (b5) - confirm file exists & is executable
		return base.ShellScriptToJob(qfs.NewMemfileBytes(p.Name, nil), p.Periodicity, nil)
	}

	var ref repo.DatasetRef
	if ref, err = repo.ParseDatasetRef(p.Name); err != nil {
		return
	}
	if err = repo.CanonicalizeDatasetRef(m.inst.Repo(), &ref); err != nil {
		return
	}
	if err = base.ReadDataset(m.inst.Repo(), &ref); err != nil {
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

	return base.DatasetToJob(ref.Dataset, p.Periodicity, o)
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

// ServiceStatus reports status of the cron daemon
func (m *UpdateMethods) ServiceStatus(in *bool, out *ServiceStatus) error {
	return fmt.Errorf("not finished")
}

// UpdateServiceStart starts the update service
func UpdateServiceStart(ctx context.Context, repoPath string, updateCfg *config.Update, deamonize bool) error {
	if updateCfg == nil {
		updateCfg = config.DefaultUpdate()
	}

	return update.Start(ctx, repoPath, updateCfg, deamonize)
}

// ServiceStart ensures the scheduler is running
func (m *UpdateMethods) ServiceStart(in, out *bool) error {
	local, ok := m.inst.cron.(*cron.Cron)
	if !ok {
		return fmt.Errorf("service already running")
	}

	ctx := context.Background()
	if err := local.Start(ctx); err != nil {
		return err
	}
	return nil
}

// ServiceStop halts the scheduler
func (m *UpdateMethods) ServiceStop(in, out *bool) error {
	// TODO (b5):
	return fmt.Errorf("not finished")
}

// ServiceRestart uses shell commands to restart the scheduler service
func (m *UpdateMethods) ServiceRestart(in, out *bool) error {
	// TODO (b5):
	return fmt.Errorf("not finished")
}

// Run advances a dataset to the latest known version from either a peer or by
// re-running a transform in the peer's namespace
func (m *UpdateMethods) Run(p *Job, res *repo.DatasetRef) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("UpdateMethods.Run", p, res)
	}

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
		err = m.runDatasetUpdate(params, res)

	case cron.JTShellScript:
		return base.JobToCmd(m.inst.streams, p).Run()

	default:
		return fmt.Errorf("unrecognized update type: %s", p.Type)
	}

	if err != nil {
		return err
	}

	// TODO (b5): expand event logging interface to support storing additional details
	return m.inst.Repo().LogEvent(repo.ETCronJobRan, *res)
}

func (m *UpdateMethods) runDatasetUpdate(p *SaveParams, res *repo.DatasetRef) error {
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
		*res, err = actions.UpdateRemoteDataset(m.inst.Node(), &ref, true)
		return err
	}

	// default to recalling transform scripts for local updates
	// TODO (b5): not sure if this should be here or in client libraries
	if p.Recall == "" {
		p.Recall = "tf"
	}

	dsr := NewDatasetRequests(m.inst.node, m.inst.rpc)
	return dsr.Save(p, res)
}
