package lib

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/cron"
	"github.com/qri-io/qri/repo"
)

// NewUpdateMethods creates a configuration handle from an instance
func NewUpdateMethods(inst *Instance) *UpdateMethods {
	return &UpdateMethods{inst: inst}
}

// UpdateMethods enapsulates logic for scheduled updates
type UpdateMethods struct {
	inst        *Instance
	ScriptsPath string
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

// Schedule
func (m *UpdateMethods) Schedule(in *ScheduleParams, out *cron.Job) (err error) {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("UpdateMethods.Schedule", in, out)
	}

	var (
		job *cron.Job
		// this context is scoped to the scheduling request. currently not cancellable
		// because our lib methods don't accept a context themselves
		ctx = context.Background()
	)

	if possibleShellScript(in.Name) {
		job, err = m.inst.cron.ScheduleShellScript(ctx, qfs.NewMemfileBytes(in.Name, nil), in.Periodicity, nil)
		*out = *job
		return
	}

	var ref repo.DatasetRef
	if ref, err = repo.ParseDatasetRef(in.Name); err != nil {
		return
	}
	if err = repo.CanonicalizeDatasetRef(m.inst.Repo(), &ref); err != nil {
		return err
	}
	if err = base.ReadDataset(m.inst.Repo(), &ref); err != nil {
		return err
	}

	// TODO (b5) - parse update options & submit them here
	job, err = m.inst.cron.ScheduleDataset(ctx, ref.Dataset, in.Periodicity, nil)
	return
}

// TODO (b5) - deal with platforms that don't use '.sh' as a script extension (windows?)
func possibleShellScript(path string) bool {
	return filepath.Ext(path) == ".sh"
}

func (m *UpdateMethods) Unschedule(name *string, unscheduled *bool) error {
	return fmt.Errorf("not finished")
}

func (m *UpdateMethods) List(name *string, unscheduled *bool) error {
	return fmt.Errorf("not finished")
}

func (m *UpdateMethods) Log(name *string, unscheduled *bool) error {
	return fmt.Errorf("not finished")
}

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

func (m *UpdateMethods) StopService(in, out *bool) error {
	return fmt.Errorf("not finished")
}

func (m *UpdateMethods) RestartService(in, out *bool) error {
	return nil
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
		runner := cron.LocalShellScriptRunner(m.ScriptsPath)
		err = runner(m.inst.ctx, m.inst.streams, p)
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

	// if !base.InLocalNamespace(m.inst.Repo(), &ref) {
	// 	*res, err = actions.UpdateRemoteDataset(m.node, &ref, true)
	// 	return err
	// }

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
		var inst *Instance
		inst, err = newInst(ctx, streams)
		if err != nil {
			return err
		}

		m := NewUpdateMethods(inst)
		m.ScriptsPath = scriptsPath
		res := &repo.DatasetRef{}
		return m.Run(job, res)
	}
}
