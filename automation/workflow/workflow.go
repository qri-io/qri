package workflow

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	golog "github.com/ipfs/go-log"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/iso8601"
	"github.com/qri-io/qri/dsref"
)

var (
	log = golog.Logger("workflow")
	// NowFunc is an overridable function for getting datestamps
	NowFunc = time.Now
)

// Type is a type for distinguishing between two different kinds of WorkflowSet
// Type should be used as a shorthand for defining how to execute a workflow
type Type string

const (
	// JTDataset indicates a workflow that RunInfoSet "qri update" on a dataset specified
	// by Workflow Name. The workflow periodicity is determined by the specified dataset's
	// Meta.AccrualPeriodicity field. LastRun should closely match the datasets's
	// latest Commit.Timestamp value
	JTDataset Type = "dataset"
	// JTShellScript represents a shell script to be run locally, which might
	// update one or more datasets. A non-zero exit code from shell script
	// indicates the workflow failed to execute properly
	JTShellScript Type = "shell"
)

// Status enumerates all possible execution states of a workflow.
type Status string

const (
	// StatusRunning indicates a workflow is currently executing
	StatusRunning = Status("running")
	// StatusSucceeded indicates a workflow has completed without error
	StatusSucceeded = Status("succeeded")
	// StatusFailed indicates a workflow completed & exited when an unexpected error
	// occured
	StatusFailed = Status("failed")
	// StatusNoChange indicates a workflow completed, but no changes occured
	StatusNoChange = Status("unchanged")
)

const (
	// WorkflowMulticodecType is a CID prefix for cron.Workflow content
	// identifiers
	// TODO(b5) - using a dummy codec number for now. Pick a real one!
	WorkflowMulticodecType = 2000
	// multihashCodec defines the hashing algorithm this package uses when
	// calculating identifiers
	// multihashCodec = multihash.BLAKE2B_MIN + 256
	multihashCodec = multihash.SHA2_256
)

// GenerateWorkflowID returns CID string with a CronWorkflowCodecType prefix
func GenerateWorkflowID() string {
	return uuid.New().String()
}

// zero is a "constant" representing an empty repeating interval
// TODO (b5) - add a IsZero methods to iso8601 structs
var zero iso8601.RepeatingInterval

// Workflow represents a "cron workflow" that can be scheduled for repeated execution at
// a specified Periodicity (time interval)
type Workflow struct {
	ID string `json:"id"` // CID string0
	// TODO (ramfox): `DatasetID` is currently expected to be the `username/name` combination
	// when the infrastructure supports it, we want to switch this over to the dataset's `InitID`
	DatasetID string     `json:"datasetID"`         // dataset identifier
	OwnerID   string     `json:"ownerID"`           // user that created this workflow
	Name      string     `json:"name"`              // human dataset name eg: "b5/world_bank_population"
	Created   *time.Time `json:"created"`           // date workflow was created
	RunCount  int        `json:"runCount"`          // number of times this workflow has been run
	Options   Options    `json:"options,omitempty"` // workflow configuration

	Disabled    bool       `json:"disabled"`    // if true, workflow will not generate new run starts
	LatestStart *time.Time `json:"latestStart"` // time workflow last started,
	LatestEnd   *time.Time `json:"latestEnd"`   // time workflow last finished, nil if currently running
	Status      Status     `json:"status"`      // the status of the workflow,  "running", "failed", "succeeded", "" for a manual run

	Triggers   Triggers `json:"triggers"`             // things that can initiate a run
	CurrentRun *RunInfo `json:"currentRun,omitempty"` // optional currently executing run
	OnComplete Hooks    `json:"onComplete"`           // things to do after a run executes

	VersionInfo dsref.VersionInfo `json:"versionInfo"` // optional versionInfo of DatasetID field

	Type Type `json:"type"` // distinguish run type
}

// NewCronWorkflow constructs a workflow pointer with a cron trigger
func NewCronWorkflow(name, ownerID, datasetID string, periodicityString string) (*Workflow, error) {
	p, err := iso8601.ParseRepeatingInterval(periodicityString)
	if err != nil {
		return nil, err
	}

	id := GenerateWorkflowID()

	t := NowFunc()
	return &Workflow{
		ID:        id,
		OwnerID:   ownerID,
		DatasetID: datasetID,
		Name:      name,
		Created:   &t,
		Triggers: Triggers{
			NewCronTrigger(id, t, p),
		},

		Type: JTDataset,
	}, nil
}

// Complete rounds out the Workflow after a dataset has been created
func (workflow *Workflow) Complete(ds *dsref.Ref, ownerID string) error {
	workflow.Name = ds.Human()
	//TODO (arqu): expand this as this version info is very shallow
	workflow.VersionInfo = ds.VersionInfo()
	workflow.Type = JTDataset
	workflow.OwnerID = ownerID
	return nil
}

// CompareWorkflows returns a string diff of the two given workflows
func CompareWorkflows(a, b *Workflow) string {
	return cmp.Diff(a, b, cmp.Comparer(CompareDurations))
}

// CompareDurations can be used in cmp.Comparer to create a cmp.Option that allows
// a cmp.Diff of `iso8601.Duration`s
func CompareDurations(x, y iso8601.Duration) bool {
	return x.String() == y.String()
}

// Info is a simplified data structure that can be built from a Workflow
// It is primarily used for the `workflow/list` endpoint
type Info struct {
	dsref.VersionInfo
	ID          string     `json:"id"` // CID string
	LatestStart *time.Time `json:"latestStart"`
	LatestEnd   *time.Time `json:"latestEnd"`
	Status      Status     `json:"status"`
}

// Info returns a `workflow.Info` from a `Workflow`
func (workflow *Workflow) Info() *Info {
	return &Info{
		VersionInfo: workflow.VersionInfo,
		ID:          workflow.ID,
		LatestStart: workflow.LatestStart,
		LatestEnd:   workflow.LatestEnd,
		Status:      workflow.Status,
	}
}

// Advance creates a new run, increments the run count, and sets the next
// execution wall, and adjusts the Status and LatestStart of the workflow
func (workflow *Workflow) Advance(triggerID string) (err error) {
	workflow.CurrentRun, err = NewRunInfo(workflow.ID, workflow.RunCount+1)
	if err != nil {
		return err
	}
	workflow.RunCount++

	workflow.LatestStart = workflow.CurrentRun.Start
	workflow.LatestEnd = nil
	workflow.Status = StatusRunning

	if triggerID != "" {
		for _, t := range workflow.Triggers {
			if t.Info().ID == triggerID {
				t.Advance(workflow)
			}
		}
	}
	return nil
}

// Copy creates a copy of a workflow
func (workflow *Workflow) Copy() *Workflow {
	cp := &Workflow{
		ID:          workflow.ID,
		DatasetID:   workflow.DatasetID,
		OwnerID:     workflow.OwnerID,
		Name:        workflow.Name,
		Created:     workflow.Created,
		RunCount:    workflow.RunCount,
		Disabled:    workflow.Disabled,
		LatestStart: workflow.LatestStart,
		LatestEnd:   workflow.LatestEnd,
		Status:      workflow.Status,
		Triggers:    workflow.Triggers,
		OnComplete:  workflow.OnComplete,
		VersionInfo: workflow.VersionInfo,
		Type:        workflow.Type,
	}

	if workflow.CurrentRun != nil {
		cp.CurrentRun = workflow.CurrentRun.Copy()
	}
	if workflow.Options != nil {
		cp.Options = workflow.Options
	}

	return cp
}

// WorkflowSet is a collection of Workflows that implements the sort.Interface,
// sorting a list of WorkflowSet in reverse-chronological-then-alphabetical order
type WorkflowSet struct {
	set []*Workflow
}

// NewWorkflowSet constructs a workflow set.
func NewWorkflowSet() *WorkflowSet {
	return &WorkflowSet{}
}

func (js WorkflowSet) Len() int { return len(js.set) }
func (js WorkflowSet) Less(i, j int) bool {
	return lessNilTime(js.set[i].LatestStart, js.set[j].LatestStart)
}
func (js WorkflowSet) Swap(i, j int) { js.set[i], js.set[j] = js.set[j], js.set[i] }

func (js *WorkflowSet) Add(j *Workflow) {
	if js == nil {
		*js = WorkflowSet{set: []*Workflow{j}}
		return
	}

	for i, workflow := range js.set {
		if workflow.ID == j.ID {
			js.set[i] = j
			return
		}
	}
	js.set = append(js.set, j)
	sort.Sort(js)
}

func (js *WorkflowSet) Remove(id string) (removed bool) {
	for i, workflow := range js.set {
		if workflow.ID == id {
			if i+1 == len(js.set) {
				js.set = js.set[:i]
				return true
			}

			js.set = append(js.set[:i], js.set[i+1:]...)
			return true
		}
	}
	return false
}

// MarshalJSON serializes WorkflowSet to an array of Workflows
func (js WorkflowSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(js.set)
}

// UnmarshalJSON deserializes from a JSON array
func (js *WorkflowSet) UnmarshalJSON(data []byte) error {
	set := []*Workflow{}
	if err := json.Unmarshal(data, &set); err != nil {
		return err
	}
	js.set = set
	return nil
}

func lessNilTime(a, b *time.Time) bool {
	if a == nil && b != nil {
		return true
	} else if a != nil && b == nil {
		return false
	} else if a == nil && b == nil {
		return false
	}
	return a.After(*b)
}

// OptionsType describes the type of Workflow
type OptionsType string

const (
	// OTDataset represents a Workflow related to running a dataset
	OTDataset OptionsType = "dataset"
	// OTShell represents a Workflow related to running a shell script
	OTShell OptionsType = "shell"
)

// Options is an interface for workflow options
type Options interface {
}

// ShellScriptOptions encapsulates options for running a shell script cron workflow
type ShellScriptOptions struct {
	// none yet.
}

// DatasetOptions encapsulates options passed to `qri save`
// TODO (b5) - we should contribute flexbuffer support for golang & remove this entirely
type DatasetOptions struct {
	Title     string
	Message   string
	Recall    string
	BodyPath  string
	FilePaths []string

	Publish             bool
	Strict              bool
	Force               bool
	ConvertFormatToPrev bool
	ShouldRender        bool

	Config  map[string]string
	Secrets map[string]string
}
