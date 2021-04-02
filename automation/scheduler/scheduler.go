package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/event"
)

var (
	log = golog.Logger("sched")
	// DefaultCheckInterval is the frequency cron will check all stored workflows
	// for scheduled updates without any additional configuration. Qri recommends
	// not running updates more than once an hour for performance and storage
	// consumption reasons, making a check every second reasonable
	DefaultCheckInterval = time.Second
	// NowFunc is an overridable function for getting datestamps
	NowFunc = time.Now
)

func init() {
	golog.SetLogLevel("sched", "debug")
}

// Scheduler is the generic interface for the Cron Scheduler
type Scheduler interface {
	Start(ctx context.Context) error
	// ListWorkflows lists currently scheduled workflows
	ListWorkflows(ctx context.Context, offset, limit int) ([]*workflow.Workflow, error)
	// WorkflowForName gets a workflow by it's name (which often matches the dataset name)
	WorkflowForName(ctx context.Context, name string) (*workflow.Workflow, error)
	// Workflow gets a single scheduled workflow by workflow identifier
	Workflow(ctx context.Context, id string) (*workflow.Workflow, error)
	// WorkflowForDataset gets a single scheduled workflow by dataset identifier
	WorkflowForDataset(ctx context.Context, id string) (*workflow.Workflow, error)
	// RunInfos gives a log of executed workflows
	RunInfos(ctx context.Context, offset, limit int) ([]*workflow.RunInfo, error)
	// GetRunInfo returns a single executed workflow by workflow.LogName
	GetRunInfo(ctx context.Context, id string, runNumber int) (*workflow.RunInfo, error)
	// // RunLogFile returns a reader for a file at the given name
	// RunLogFile(ctx context.Context, id string, runNumber int) (io.ReadCloser, error)

	// Schedule adds a workflow to the scheduler for execution
	Schedule(ctx context.Context, w *workflow.Workflow) error
	// Unschedule removes a workflow from the scheduler
	Unschedule(ctx context.Context, id string) error
	// UpdateWorkflow
	UpdateWorkflow(ctx context.Context, w *workflow.Workflow) error
}

// RunWorkflowFunc is a function for executing a workflow. Cron takes care of scheduling
// workflow execution, and delegates the work of executing a workflow to a RunWorkflowFunc
// implementation.
type RunWorkflowFunc func(ctx context.Context, streams ioes.IOStreams, w *workflow.Workflow) error

// RunWorkflowFactory is a function that returns a runner
type RunWorkflowFactory func(ctx context.Context) (runner RunWorkflowFunc)

// Cron coordinates the scheduling of running workflows at specified periodicities
// (intervals) with a provided workflow runner function
type Cron struct {
	pub      event.Publisher
	store    workflow.Store
	interval time.Duration
	factory  RunWorkflowFactory

	runLk sync.Mutex
}

// assert Cron is a Scheduler at compile time
var _ Scheduler = (*Cron)(nil)

// NewCron creates a Cron with the default check interval
func NewCronScheduler(store workflow.Store, factory RunWorkflowFactory, pub event.Publisher) Scheduler {
	return NewCronSchedulerInterval(store, factory, pub, DefaultCheckInterval)
}

// NewCronInterval creates a Cron with a check interval
func NewCronSchedulerInterval(store workflow.Store, factory RunWorkflowFactory, pub event.Publisher, checkInterval time.Duration) Scheduler {
	return &Cron{
		pub:      pub,
		store:    store,
		interval: checkInterval,
		factory:  factory,
	}
}

// ListWorkflows proxies to the schedule store for reading workflows
func (c *Cron) ListWorkflows(ctx context.Context, offset, limit int) ([]*workflow.Workflow, error) {
	return c.store.ListWorkflows(ctx, offset, limit)
}

// ListWorkflowsByStatus proxies to the scheduler store for reading workflows by status
// returns workflows is reverse chronological order by `LatestStart`
func (c *Cron) ListWorkflowsByStatus(ctx context.Context, status workflow.Status, offset, limit int) ([]*workflow.Workflow, error) {
	return c.store.ListWorkflowsByStatus(ctx, status, offset, limit)
}

// ListRunningCollection converts currently running workflows into `WorkflowInfo`s
func (c *Cron) ListRunningCollection(ctx context.Context, offset, limit int) ([]*workflow.Info, error) {
	ws, err := c.ListWorkflowsByStatus(context.Background(), workflow.StatusRunning, offset, limit)
	if err != nil {
		return nil, err
	}
	wis := []*workflow.Info{}
	for _, w := range ws {
		wis = append(wis, w.Info())
	}
	return wis, nil
}

// WorkflowForName gets a workflow by it's name (which often matches the dataset name)
func (c *Cron) WorkflowForName(ctx context.Context, name string) (*workflow.Workflow, error) {
	return c.store.GetWorkflowByName(ctx, name)
}

// workflow.Workflow proxies to the schedule store for reading a workflow by name
func (c *Cron) Workflow(ctx context.Context, id string) (*workflow.Workflow, error) {
	return c.store.GetWorkflow(ctx, id)
}

// WorkflowForDataset gets a single scheduled workflow by dataset identifier
func (c *Cron) WorkflowForDataset(ctx context.Context, id string) (*workflow.Workflow, error) {
	return c.store.GetWorkflowByDatasetID(ctx, id)
}

// RunInfos returns a list of workflows that have been executed
func (c *Cron) RunInfos(ctx context.Context, offset, limit int) ([]*workflow.RunInfo, error) {
	return c.store.ListRunInfos(ctx, offset, limit)
}

// GetRunInfo gives a specific Run by datasetID and run Number
func (c *Cron) GetRunInfo(ctx context.Context, datasetID string, runNumber int) (*workflow.RunInfo, error) {
	return nil, fmt.Errorf("not finished: service get run by datasetID and runNumber")
}

// Start initiates the check loop, looking for updates to execute once at every
// iteration of the configured check interval.
// Start blocks until the passed context completes
func (c *Cron) Start(ctx context.Context) error {
	log.Debug("starting scheduler")
	check := func(ctx context.Context) {
		now := NowFunc()
		ctx, cleanup := context.WithCancel(ctx)
		defer cleanup()

		workflows, err := c.store.ListWorkflows(ctx, 0, -1)
		if err != nil {
			log.Errorf("getting workflows from store: %s", err)
			return
		}

		run := []*workflow.Workflow{}
		trigger := []workflow.Trigger{}
		for _, w := range workflows {
			if w.Disabled {
				log.Debugf("workflow disabled: %q", w.ID)
				continue
			}
			for _, t := range w.Triggers {
				if t.Info().Disabled {
					log.Debugf("trigger disabled: %q", t.Info().ID)
					continue
				}
				// TODO (arqu): handle other trigger types
				switch t.Info().Type {
				case workflow.TTCron:
					crn := t.(*workflow.CronTrigger)
					if crn.NextRunStart != nil && now.After(*crn.NextRunStart) {
						run = append(run, w)
						trigger = append(trigger, t)
					}
				default:
					log.Debugf("trigger type not implemented: %q", t.Info().Type)
				}
			}
		}

		if len(run) > 0 {
			log.Debugw("running workflows", "workflowCount", len(workflows), "runCount", len(run))
			for i, w := range run {
				// TODO (b5) - if we want things like per-workflow timeout, we should create
				// a new workflow-scoped context here
				c.RunWorkflow(ctx, w, trigger[i].Info().ID)
			}
		}
	}

	t := time.NewTicker(c.interval)
	for {
		select {
		case <-t.C:
			// TODO(b5): running these checks in a goroutine kicks off major issues when
			// running dataset updates. If multiple updates are scheduled in a tight loop
			// two writes can kick off at the same time, causing all sorts of undefiend
			// behaviour. We can only restore this goroutine once a qri instance is deemed
			// safe for concurrent use
			// go check(ctx)
			check(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

// RunWorkflow runs and updates the workflow
// It emits `ETWorkflowStarted` and `ETWorkflowCompleted` events with the updated
// workflow events. It is not responsible for storing the resultant workflow.
func (c *Cron) RunWorkflow(ctx context.Context, w *workflow.Workflow, triggerID string) {
	c.runLk.Lock()
	defer c.runLk.Unlock()

	runner := c.factory(ctx)

	log.Debugf("run workflow: %s", w.Name)
	if err := w.Advance(triggerID); err != nil {
		log.Debug(err)
	}

	go func(j *workflow.Workflow) {
		if err := c.pub.Publish(ctx, workflow.ETWorkflowStarted, j); err != nil {
			log.Debug(err)
		}
	}(w.Copy())

	streams := ioes.NewDiscardIOStreams()
	if lfc, ok := c.store.(workflow.LogFileCreator); ok {
		if file, logPath, err := lfc.CreateLogFile(w); err == nil {
			log.Debugf("using log file: %s", logPath)
			defer file.Close()
			streams = ioes.NewIOStreams(nil, file, file)
			w.CurrentRun.LogFilePath = logPath
		}
	}

	if err := runner(ctx, streams, w); err != nil {
		if errors.Is(err, dsfs.ErrNoChanges) {
			log.Debugf("run workflow: %s no changes", w.Name)
			w.CurrentRun.Error = ""
			w.Status = workflow.StatusNoChange
		} else {
			log.Errorf("run workflow: %s error: %s", w.Name, err.Error())
			w.CurrentRun.Error = err.Error()
			w.Status = workflow.StatusFailed
		}
	} else {
		log.Debugf("run workflow: %s success", w.Name)
		w.CurrentRun.Error = ""
		w.Status = workflow.StatusSucceeded
	}
	now := NowFunc()
	w.CurrentRun.Stop = &now
	w.LatestEnd = &now

	go func(j *workflow.Workflow) {
		if err := c.pub.Publish(ctx, workflow.ETWorkflowCompleted, j); err != nil {
			log.Debug(err)
		}
	}(w.Copy())
}

// Schedule adds a workflow to the cron scheduler
func (c *Cron) Schedule(ctx context.Context, w *workflow.Workflow) (err error) {
	// ensure IDs are aligned to avoid double workflows/scheduling
	wf, err := c.store.GetWorkflowByDatasetID(ctx, w.DatasetID)
	if err == nil && (w.ID == "" || w.ID != wf.ID) {
		return fmt.Errorf("schedule: bad workflow ID")
	}

	// if we're scheduling a workflow, it means it's enabled
	w.Disabled = false

	for _, t := range w.Triggers {
		if t.Info().Disabled {
			log.Debugf("trigger disabled: %q", t.Info().ID)
			continue
		}
		// TODO (arqu): handle other trigger types
		switch t.Info().Type {
		case workflow.TTCron:
			crn := t.(*workflow.CronTrigger)
			if crn.NextRunStart == nil {
				crn.NextRunStart = crn.NextExecutionWall()
			}
		default:
			log.Debugf("trigger type not implemented: %q", t.Info().Type)
		}
	}

	if err := c.store.PutWorkflow(ctx, w); err != nil {
		return err
	}

	go func(j *workflow.Workflow) {
		if err := c.pub.Publish(ctx, workflow.ETWorkflowScheduled, j); err != nil {
			log.Debug(err)
		}
	}(w.Copy())

	return nil
}

// Unschedule removes a workflow from the cron scheduler, cancelling any future
// workflow executions
func (c *Cron) Unschedule(ctx context.Context, id string) error {
	// TODO(arqu): this should also remove associated runs
	if err := c.store.DeleteWorkflow(ctx, id); err != nil {
		return err
	}
	go func() {
		if err := c.pub.Publish(ctx, workflow.ETWorkflowUnscheduled, id); err != nil {
			log.Debug(err)
		}
	}()

	return nil
}

// UpdateWorkflow updates the workflow without triggering a run
func (c *Cron) UpdateWorkflow(ctx context.Context, w *workflow.Workflow) error {
	if w == nil {
		return fmt.Errorf("workflow is nil")
	}
	if w.ID == "" {
		return fmt.Errorf("bad workflow ID")
	}
	wf, err := c.store.GetWorkflow(ctx, w.ID)
	if err != nil {
		return err
	}

	wf.Disabled = w.Disabled
	wf.Triggers = w.Triggers
	wf.OnComplete = w.OnComplete

	if err := c.store.PutWorkflow(ctx, wf); err != nil {
		return err
	}
	go func() {
		if err := c.pub.Publish(ctx, workflow.ETWorkflowUpdated, wf.ID); err != nil {
			log.Debug(err)
		}
	}()

	return nil
}

// // ListCollection returns a union of datasets and workflows in the form of `Info`s
// // TODO (ramfox): add pagination by timestamp
// func (c *Cron) ListCollection(ctx context.Context, inst *lib.Instance, before, after time.Time) ([]*workflow.Info, error) {
// 	m := inst.Dataset()
// 	// TODO (ramfox): for now we are fetching everything.
// 	p := &lib.ListParams{
// 		Offset: 0,
// 		Limit:  100000000000,
// 	}

// 	// TODO (ramfox): when we add in pagination, we should be using `after` and `before`
// 	// as our metrics. We should use those to search for the correct interval of datasets
// 	// and the correct interval of workflows

// 	// TODO (ramfox): goal is eventually to get version infos list in reverse
// 	// chronological order by activity
// 	// However dataset list does not have the ability to sort in a specified way
// 	vis := []dsref.VersionInfo{}
// 	fetchNext := true
// 	for fetchNext {
// 		v, err := m.List(ctx, p)
// 		if err != nil {
// 			log.Errorf("error getting datasets: %w", err)
// 			return nil, fmt.Errorf("error getting datasets: %w", err)
// 		}
// 		vis = append(vis, v...)
// 		if len(v) < p.Limit {
// 			fetchNext = false
// 		}
// 		p.Offset++
// 	}

// 	// -1 limit returns all workflows
// 	ws, err := c.store.ListWorkflows(ctx, 0, -1)
// 	if err != nil {
// 		log.Errorf("error getting workflows: %w", err)
// 		return nil, fmt.Errorf("error getting workflows: %w", err)
// 	}

// 	wiMap := map[string]*workflow.Workflow{}

// 	for _, w := range ws {
// 		wiMap[w.DatasetID] = w
// 	}

// 	wis := []*workflow.Info{}
// 	for _, vi := range vis {
// 		// DatasetID is currently `username/name`
// 		viID := vi.Alias()
// 		w, ok := wiMap[viID]
// 		if ok {
// 			w.VersionInfo = vi
// 			wis = append(wis, w.Info())
// 			continue
// 		}
// 		// TODO (ramfox): HACK - because frontend has no concept of identity yet
// 		// all workflows created by the frontend are sent with `Username='me'`
// 		w, ok = wiMap[fmt.Sprintf("me/%s", vi.Name)]
// 		if ok {
// 			w.VersionInfo = vi
// 			wis = append(wis, w.Info())
// 			continue
// 		}
// 		// TODO (ramfox): using the dataset alias as the workflow id for now
// 		// this should be replaced with the the `InitID`, once that is surfaced
// 		// in the `VersionInfo`
// 		wis = append(wis, &workflow.Info{VersionInfo: vi, ID: vi.Alias()})
// 	}

// 	sort.Slice(wis, func(i, j int) bool {
// 		// sort by commit time in reverse chronological order
// 		// TODO (ramfox): when `activity time` is surfaced, we would prefer to sort
// 		// by that metric
// 		return wis[i].CommitTime.After(wis[j].CommitTime)
// 	})

// 	return wis, nil
// }

// DeployParams represents what we need in order to deploy a workflow
type DeployParams struct {
	Apply     bool               `json:"apply"`
	Workflow  *workflow.Workflow `json:"workflow"`
	Transform *dataset.Transform `json:"transform"`
}

// DeployResponse is what we return when we first deploy a workflow
type DeployResponse struct {
	RunID    string             `json:"runID"`
	Workflow *workflow.Workflow `json:"workflow"`
}

// // Deploy takes a workflow and transform and returns a runid and workflow
// // It applys a transform to a specified dataset and schedules the workflow
// func (c *Cron) Deploy(ctx context.Context, inst *lib.Instance, p *DeployParams) (*DeployResponse, error) {
// 	if p.Workflow == nil {
// 		return nil, fmt.Errorf("deploy: workflow not set")
// 	}
// 	if p.Workflow.DatasetID == "" {
// 		return nil, fmt.Errorf("deploy: DatasetID not set")
// 	}

// 	wf := p.Workflow
// 	bus := inst.Bus()

// 	newWorkflow := true
// 	if _, err := c.Workflow(ctx, wf.ID); err == nil {
// 		newWorkflow = false
// 	}
// 	if wf.ID == "" && wf.OwnerID != "" && wf.DatasetID != "" {
// 		wf.ID = workflow.GenerateWorkflowID()
// 	}

// 	now := time.Now()
// 	if newWorkflow {
// 		wf.Created = &now
// 	}
// 	wf.LatestStart = &now

// 	go func() {
// 		if err := bus.PublishID(ctx, workflow.ETWorkflowDeployStarted, wf.ID, wf.Info()); err != nil {
// 			log.Debugw("async event error", "evt", workflow.ETWorkflowDeployStarted, "workflowID", wf.ID, "err", err)
// 		}
// 	}()
// 	defer func() {
// 		go func() {
// 			if err := bus.PublishID(ctx, workflow.ETWorkflowDeployStopped, wf.ID, wf.Info()); err != nil {
// 				log.Debugw("async event error", "evt", workflow.ETWorkflowDeployStopped, "workflowID", wf.ID, "err", err)
// 			}
// 		}()
// 	}()

// 	dsm := inst.Dataset()
// 	saveP := &lib.SaveParams{
// 		Ref: wf.DatasetID, // currently the DatasetID is the Ref
// 		Dataset: &dataset.Dataset{
// 			Transform: p.Transform,
// 		},
// 		Apply: p.Apply,
// 		// Wait: false,
// 	}
// 	log.Debugw("deploying dataset", "datasetID", saveP.Ref)
// 	res, err := dsm.Save(ctx, saveP)
// 	if err != nil {
// 		if errors.Is(err, dsfs.ErrNoChanges) {
// 			err = nil
// 		} else {
// 			log.Errorw("deploy save dataset", "error", err)
// 			return nil, err
// 		}
// 	}

// 	now = NowFunc()
// 	wf.LatestEnd = &now
// 	wf.RunCount++
// 	wf.Status = workflow.StatusSucceeded

// 	if newWorkflow {
// 		ref := &dsref.Ref{
// 			Username: res.Peername,
// 			Name:     res.Name,
// 		}
// 		wf.Complete(ref, inst.GetConfig().Profile.ID)
// 	}

// 	err = c.Schedule(ctx, wf)
// 	if err != nil {
// 		log.Errorw("deploy scheduling", "error", err)
// 	}

// 	return &DeployResponse{
// 		Workflow: wf,
// 	}, err
// }

// Undeploy takes a workflow and removes it from the scheduler
func (c *Cron) Undeploy(ctx context.Context, workflowID string) error {
	err := c.Unschedule(ctx, workflowID)
	if err != nil {
		log.Errorw("undeploy unscheduling", "error", err)
	}
	return err
}
