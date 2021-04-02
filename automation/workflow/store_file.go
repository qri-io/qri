package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"sync"

	"github.com/qri-io/qri/event"
)

// fileStore is a store implementation that writes to a file of JSON bytes.
// fileStore is safe for concurrent use
type fileStore struct {
	path string

	lock             sync.Mutex
	workflows        *WorkflowSet
	workflowRunInfos map[string]*RunInfoSet
	runs             *RunInfoSet
}

// compile-time assertion that fileStore is a Store
var _ Store = (*fileStore)(nil)

// NewFileStore creates a workflow store that persists to a file
func NewFileStore(path string, bus event.Bus) (Store, error) {
	s := &fileStore{
		path:             path,
		workflows:        NewWorkflowSet(),
		workflowRunInfos: map[string]*RunInfoSet{},
		runs:             NewRunInfoSet(),
	}

	subscribe(s, bus)

	log.Debugw("creating file store")
	return s, s.loadFromFile()
}

// ListWorkflows lists workflows currently in the store
func (s *fileStore) ListWorkflows(ctx context.Context, offset, limit int) ([]*Workflow, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if limit < 0 {
		limit = len(s.workflows.set)
	}

	workflows := make([]*Workflow, 0, limit)
	for i, workflow := range s.workflows.set {
		if i < offset {
			continue
		} else if len(workflows) == limit {
			break
		}

		workflows = append(workflows, workflow)
	}
	return workflows, nil
}

// ListWorkflowsByStatus lists workflows filtered by status and ordered in reverse
// chronological order by `LatestStart`
func (s *fileStore) ListWorkflowsByStatus(ctx context.Context, status Status, offset, limit int) ([]*Workflow, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	workflows := make([]*Workflow, 0, len(s.workflows.set))

	for _, workflow := range s.workflows.set {
		if workflow.Status == status {
			log.Debugf("workflow %s has correct status", workflow.ID)
			workflows = append(workflows, workflow)
		}
	}

	if offset > len(workflows) {
		return []*Workflow{}, nil
	}

	sort.Slice(workflows, func(i, j int) bool {
		if workflows[j].LatestStart == nil {
			return false
		}
		if workflows[i].LatestStart == workflows[j].LatestStart {
			return workflows[i].Name < workflows[j].Name
		}
		return workflows[i].LatestStart.After(*(workflows[j].LatestStart))
	})

	if limit < 0 {
		limit = len(workflows)
	}

	if offset+limit > len(workflows) {
		return workflows[offset:], nil
	}

	return workflows[offset:limit], nil
}

// ListRunInfos returns a slice of `RunInfo`s
func (s *fileStore) ListRunInfos(ctx context.Context, offset, limit int) ([]*RunInfo, error) {
	if limit < 0 {
		limit = len(s.runs.set)
	}

	runs := make([]*RunInfo, 0, limit)
	for i, workflow := range s.runs.set {
		if i < offset {
			continue
		} else if len(runs) == limit {
			break
		}

		runs = append(runs, workflow)
	}

	return runs, nil
}

// GetWorkflowByName gets a workflow with the corresponding name field. usually matches
// the dataset name
func (s *fileStore) GetWorkflowByName(ctx context.Context, name string) (*Workflow, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, workflow := range s.workflows.set {
		if workflow.Name == name {
			return workflow.Copy(), nil
		}
	}
	return nil, ErrNotFound
}

// GetWorkflowByDatasetID gets a workflow with the corresponding datasetID field
func (s *fileStore) GetWorkflowByDatasetID(ctx context.Context, datasetID string) (*Workflow, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, workflow := range s.workflows.set {
		if workflow.DatasetID == datasetID {
			return workflow.Copy(), nil
		}
	}
	return nil, ErrNotFound
}

// GetWorkflow gets workflow details from the store by dataset identifier
func (s *fileStore) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, workflow := range s.workflows.set {
		if workflow.ID == id {
			return workflow.Copy(), nil
		}
	}
	return nil, ErrNotFound
}

// PutWorkflow places a workflow in the store. If the workflow name matches the name of a workflow
// that already exists, it will be overwritten with the new workflow
func (s *fileStore) PutWorkflow(ctx context.Context, workflow *Workflow) error {
	if workflow.ID == "" {
		return fmt.Errorf("ID is required")
	}
	if workflow.DatasetID == "" {
		return fmt.Errorf("DatasetID is required")
	}

	s.lock.Lock()
	s.workflows.Add(workflow)
	s.lock.Unlock()

	if workflow.CurrentRun != nil {
		if err := s.PutRunInfo(ctx, workflow.CurrentRun); err != nil {
			return err
		}
	}
	return s.writeToFile()
}

// DeleteWorkflow removes a workflow from the store by name. deleting a non-existent workflow
// won't return an error
func (s *fileStore) DeleteWorkflow(ctx context.Context, id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if removed := s.workflows.Remove(id); removed {
		return s.writeToFileNoLock()
	}
	return ErrNotFound
}

// GetRunInfo fetches a run by ID
func (s *fileStore) GetRunInfo(ctx context.Context, id string) (*RunInfo, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, r := range s.runs.set {
		if r.ID == id {
			return r.Copy(), nil
		}
	}
	return nil, ErrNotFound
}

// GetWorkflowRunInfos gets the `RunInfo`s of a specific workflow, by `Workflow.ID`
func (s *fileStore) GetWorkflowRunInfos(ctx context.Context, workflowID string, offset, limit int) ([]*RunInfo, error) {
	ris, ok := s.workflowRunInfos[workflowID]
	if !ok {
		return nil, ErrNotFound
	}

	if limit < 0 {
		return ris.set[offset:], nil
	}

	res := make([]*RunInfo, 0, limit)
	for _, run := range ris.set {
		if offset > 0 {
			offset--
			continue
		}
		if len(res) == limit {
			return res, nil
		}
		res = append(res, run)
	}
	return res, nil
}

// PutRunInfo puts a `RunInfo` into the store
func (s *fileStore) PutRunInfo(ctx context.Context, run *RunInfo) error {
	if run.ID == "" {
		return fmt.Errorf("ID is required")
	}
	if run.WorkflowID == "" {
		return fmt.Errorf("WorkflowID is required")
	}

	s.lock.Lock()
	if workflowRunInfos, ok := s.workflowRunInfos[run.WorkflowID]; ok {
		workflowRunInfos.Add(run)
	} else {
		workflowRunInfos = NewRunInfoSet()
		workflowRunInfos.Add(run)
		s.workflowRunInfos[run.WorkflowID] = workflowRunInfos
	}
	s.runs.Add(run)
	s.lock.Unlock()
	return s.writeToFile()
}

// DeleteAllWorkflowRunInfos removes all the RunInfos of a specific workflow
// TODO (ramfox): not finished
func (s *fileStore) DeleteAllWorkflowRunInfos(ctx context.Context, workflowID string) error {
	return fmt.Errorf("not finished: fileStore delete all workflow runs")
}

// DeleteAllWorkflows removes all the workflow from the filestore
// TODO (ramfox): not finished
func (s *fileStore) DeleteAlWorkflows(ctx context.Context) error {
	return fmt.Errorf("not finished: fileStore delete all workflows")
}

// const logsDirName = "logfiles"

// // CreateLogFile creates a log file in the specified logs directory
// func (s *fileStore) CreateLogFile(j *Workflow) (f io.WriteCloser, path string, err error) {
// 	s.lock.Lock()
// 	defer s.lock.Unlock()

// 	var logsDir string
// 	if logsDir, err = s.logsDir(); err != nil {
// 		return
// 	}
// 	path = filepath.Join(logsDir, j.LogName()+".log")

// 	f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
// 	return
// }

// func (s *fileStore) logsDir() (string, error) {
// 	path := filepath.Join(filepath.Dir(s.path), logsDirName)
// 	err := os.MkdirAll(path, os.ModePerm)
// 	return path, err
// }

// // Destroy removes the path entirely
// func (s *fileStore) Destroy() error {
// 	os.RemoveAll(filepath.Join(filepath.Dir(s.path), logsDirName))
// 	return os.Remove(s.path)
// }

func (s *fileStore) loadFromFile() (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	data, err := ioutil.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		log.Debugw("fileStore loading store from file", "error", err)
		return err
	}

	state := struct {
		Workflows        *WorkflowSet
		WorkflowRunInfos map[string]*RunInfoSet
		Runs             *RunInfoSet
	}{}
	if err := json.Unmarshal(data, &state); err != nil {
		log.Debugw("fileStore deserializing from JSON", "error", err)
		return err
	}

	if state.Workflows != nil {
		s.workflows = state.Workflows
	}
	if state.WorkflowRunInfos != nil {
		s.workflowRunInfos = state.WorkflowRunInfos
	}
	if state.Runs != nil {
		s.runs = state.Runs
	}
	return nil
}

func (s *fileStore) writeToFile() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.writeToFileNoLock()
}

// Only use this when you have a surrounding lock
func (s *fileStore) writeToFileNoLock() error {
	state := struct {
		Workflows        *WorkflowSet
		WorkflowRunInfos map[string]*RunInfoSet
		Runs             *RunInfoSet
	}{
		Workflows:        s.workflows,
		WorkflowRunInfos: s.workflowRunInfos,
		Runs:             s.runs,
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.path, data, 0644)
}
