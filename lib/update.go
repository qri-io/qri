package lib

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/cron"
	"github.com/qri-io/qri/repo"
)

// NewUpdateMethods creates a configuration handle from an instance
func NewUpdateMethods(inst *Instance) *UpdateMethods {
	m := &UpdateMethods{
		inst:        inst,
		scriptsPath: filepath.Join(inst.QriPath(), "update_scripts"),
	}

	if err := os.MkdirAll(m.scriptsPath, os.ModePerm); err != nil {
		log.Error("creating update scripts directory: %s", err.Error())
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

	// TODO (b5) - options support
}

// Schedule creates a job and adds it to the scheduler
func (m *UpdateMethods) Schedule(in *ScheduleParams, out *cron.Job) (err error) {
	if base.PossibleShellScript(in.Name) {
		if err = qfs.AbsPath(&in.Name); err != nil {
			return
		}
	}

	if m.inst.rpc != nil {
		return m.inst.rpc.Call("UpdateMethods.Schedule", in, out)
	}

	job, err := m.jobFromScheduleParams(in)
	if err != nil {
		return nil
	}
	*out = *job

	// this context is scoped to the scheduling request. currently not cancellable
	// because our lib methods don't accept a context themselves
	// TODO (b5): refactor RPC communication to use context
	var ctx = context.Background()

	// TODO (b5) - parse update options & submit them here
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

	return base.DatasetToJob(ref.Dataset, p.Periodicity, nil)
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

	list, err := m.inst.cron.Jobs(ctx, p.Offset, p.Limit)
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

	var res *Job
	res, err = m.inst.cron.Job(ctx, *name)
	*job = *res

	return
}

// Log shows the history of job execution
func (m *UpdateMethods) Log(name *string, unscheduled *bool) error {
	// TODO (b5)
	return fmt.Errorf("not finished")
}

// StartService ensures the scheduler is running
func (m *UpdateMethods) StartService(in, out *bool) error {
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

// StopService halts the scheduler
func (m *UpdateMethods) StopService(in, out *bool) error {
	// TODO (b5):
	return fmt.Errorf("not finished")
}

// RestartService uses shell commands to restart the scheduler service
func (m *UpdateMethods) RestartService(in, out *bool) error {
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
		updateParams := &UpdateParams{
			Ref: p.Name,
			// TODO (b5): fill in params from job Options
		}
		*res = repo.DatasetRef{}
		err = m.runDatasetUpdate(updateParams, res)

	case cron.JTShellScript:
		runner := base.LocalShellScriptRunner(m.scriptsPath)
		err = runner(m.inst.ctx, m.inst.streams, p)

	default:
		return fmt.Errorf("unrecognized update type: %s", p.Type)
	}

	if err != nil {
		return err
	}

	// TODO (b5): expand event logging interface to support storing additional details
	return m.inst.Repo().LogEvent(repo.ETCronJobRan, *res)
}

// UpdateParams defines parameters for the Update command
// TODO (b5): I think we can merge this into SaveParams
type UpdateParams struct {
	Ref          string
	Title        string
	Message      string
	Recall       string
	Secrets      map[string]string
	Publish      bool
	DryRun       bool
	ReturnBody   bool
	ShouldRender bool
	// optional writer to have transform script record standard output to
	// note: this won't work over RPC, only on local calls
	ScriptOutput io.Writer
}

func (m *UpdateMethods) runDatasetUpdate(p *UpdateParams, res *repo.DatasetRef) error {
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

	// default to recalling transfrom scripts for local updates
	// TODO (b5): not sure if this should be here or in client libraries
	if p.Recall == "" {
		p.Recall = "tf"
	}

	saveParams := &SaveParams{
		Dataset: &dataset.Dataset{
			Name:      ref.Name,
			Peername:  ref.Peername,
			ProfileID: ref.ProfileID.String(),
			Path:      ref.Path,
			Commit: &dataset.Commit{
				Title:   p.Title,
				Message: p.Message,
			},
			Transform: &dataset.Transform{
				Secrets: p.Secrets,
			},
		},
		Recall:       p.Recall,
		Secrets:      p.Secrets,
		Publish:      p.Publish,
		DryRun:       p.DryRun,
		ReturnBody:   p.ReturnBody,
		ScriptOutput: p.ScriptOutput,
	}

	dsr := NewDatasetRequests(m.inst.node, m.inst.rpc)
	return dsr.Save(saveParams, res)
}

func newUpdateRunner(newInst func(ctx context.Context, streams ioes.IOStreams) (*Instance, error), scriptsPath string) cron.RunJobFunc {
	return func(ctx context.Context, streams ioes.IOStreams, job *cron.Job) (err error) {
		log.Infof("running update: %s", job.Name)
		var inst *Instance
		inst, err = newInst(ctx, streams)
		if err != nil {
			return err
		}

		m := NewUpdateMethods(inst)
		res := &repo.DatasetRef{}
		return m.Run(job, res)
	}
}
