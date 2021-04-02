package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

// RunMulticodecType is a CID prefix for scheduler.RunInfo content identifiers
// TODO(b5) - using a dummy codec number for now. Pick a real one!
const RunMulticodecType = 2200

func runID(workflowID string, created time.Time) (string, error) {
	str := fmt.Sprintf("%s.%d", workflowID, created.UnixNano())
	mh, err := multihash.Encode([]byte(str), multihashCodec)
	if err != nil {
		return "", err
	}
	return cid.NewCidV1(RunMulticodecType, multihash.Multihash(mh)).String(), nil
}

// RunInfo is a record of workflow execution
type RunInfo struct {
	ID          string     `json:"ID"`
	WorkflowID  string     `json:"workflowID"`
	Number      int        `json:"number"`
	Start       *time.Time `json:"start"`
	Stop        *time.Time `json:"stop"`
	Error       string     `json:"error,omitempty"`
	LogFilePath string     `json:"logFilePath,omitempty"`
}

// NewRunInfo constructs a run pointer
func NewRunInfo(workflowID string, number int) (*RunInfo, error) {
	created := NowFunc()
	id, err := runID(workflowID, created)
	if err != nil {
		return nil, err
	}

	return &RunInfo{
		ID:         id,
		WorkflowID: workflowID,
		Number:     number,
		Start:      &created,
	}, nil
}

// Copy allocates a new Run pointer with all fields set to the value of the
// receiver
func (r *RunInfo) Copy() *RunInfo {
	if r == nil {
		return nil
	}
	return &RunInfo{
		ID:          r.ID,
		WorkflowID:  r.WorkflowID,
		Number:      r.Number,
		Start:       r.Start,
		Stop:        r.Stop,
		Error:       r.Error,
		LogFilePath: r.LogFilePath,
	}
}

// LogName returns a canonical name string for a run that's executed and saved
// to a logging system
func (r *RunInfo) LogName() string {
	return fmt.Sprintf("%s-%d", r.WorkflowID, r.Number)
}

// RunInfoSet is a list of RunInfoSet that implements the sort.Interface, sorting a list
// of RunInfoSet in reverse-chronological-then-alphabetical order
type RunInfoSet struct {
	set []*RunInfo
}

// NewRunInfoSet constructs a RunInfoSet.
func NewRunInfoSet() *RunInfoSet {
	return &RunInfoSet{}
}

func (rs RunInfoSet) Len() int { return len(rs.set) }
func (rs RunInfoSet) Less(i, j int) bool {
	return lessNilTime(rs.set[i].Start, rs.set[j].Start)
}
func (rs RunInfoSet) Swap(i, j int) { rs.set[i], rs.set[j] = rs.set[j], rs.set[i] }

// Add inserts a RunInfo into the RunInfoSet
func (rs *RunInfoSet) Add(r *RunInfo) {
	if rs == nil {
		*rs = RunInfoSet{set: []*RunInfo{r}}
		return
	}

	for i, run := range rs.set {
		if run.ID == r.ID {
			rs.set[i] = r
			return
		}
	}
	rs.set = append(rs.set, r)
	sort.Sort(rs)
}

// Remove removes a RunInfo from the RunInfoSet by run.ID
func (rs *RunInfoSet) Remove(id string) (removed bool) {
	for i, run := range rs.set {
		if run.ID == id {
			if i+1 == len(rs.set) {
				rs.set = rs.set[:i]
				return true
			}

			rs.set = append(rs.set[:i], rs.set[i+1:]...)
			return true
		}
	}
	return false
}

// MarshalJSON serializes RunInfoSet to an array of Workflows
func (rs RunInfoSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(rs.set)
}

// UnmarshalJSON deserializes from a JSON array
func (rs *RunInfoSet) UnmarshalJSON(data []byte) error {
	set := []*RunInfo{}
	if err := json.Unmarshal(data, &set); err != nil {
		return err
	}
	rs.set = set
	return nil
}
