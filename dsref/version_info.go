package dsref

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/qri-io/dataset"
)

// VersionInfo is an aggregation of fields from a dataset version for caching &
// listing purposes. VersionInfos are typically used when showing a list of
// datasets or a list of dataset versions ("qri list" and "qri log"). Fields on
// VersionInfo are focused on being the minimum set of values required to drive
// user interfaces that list datasets.
//
// VersionInfos can also describe dataset versions that are being created or
// failed to create. In these cases the calculated VersionInfo.Path value must
// always equal the empty string.
//
// If any fields are added to this struct, keep it in sync with:
//   dscache/def.fbs       dscache
//   dscache/fill_info.go  func fillInfoForDatasets
//   repo/ref/convert.go   func ConvertToVersionInfo
// If you are considering making major changes to VersionInfo, read this
// synopsis first:
//   https://github.com/qri-io/qri/pull/1641#issuecomment-778521313
type VersionInfo struct {
	//
	// Key as a stable identifier
	//
	// InitID is derived from the logbook for the dataset
	InitID string `json:"initID,omitempty"`
	//
	// Fields from dsref.Ref
	//
	// Username of dataset owner
	Username string `json:"username,omitempty"`
	// ProfileID of dataset owner
	ProfileID string `json:"profileID,omitempty"`
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path string `json:"path,omitempty"`
	//
	// State about the dataset that can change
	//
	// If true, this dataset has published versions
	Published bool `json:"published,omitempty"`
	// If true, this reference doesn't exist locally. Only makes sense if path is set, as this
	// flag refers to specific versions, not to entire dataset histories.
	Foreign bool `json:"foreign,omitempty"`
	//
	// Meta fields
	//
	// Title from the meta structure
	MetaTitle string `json:"metaTitle,omitempty"`
	// List of themes from the meta structure, comma-separated list
	ThemeList string `json:"themeList,omitempty"`
	//
	// Structure fields
	//
	// Size of the body in bytes
	BodySize int `json:"bodySize,omitempty"`
	// Num of rows in the body
	BodyRows int `json:"bodyRows,omitempty"`
	// Format of the body, such as "csv" or "json"
	BodyFormat string `json:"bodyFormat,omitempty"`
	// Number of errors from the structure
	NumErrors int `json:"numErrors,omitempty"`
	//
	// Commit fields
	//
	// Timestamp field from the commit
	CommitTime time.Time `json:"commitTime,omitempty"`
	// Title field from the commit
	CommitTitle string `json:"commitTitle,omitempty"`
	// Message field from the commit
	CommitMessage string `json:"commitMessage,omitempty"`
	//
	// About the dataset's history and location
	//
	// Number of versions that the dataset has
	NumVersions int `json:"numVersions,omitempty"`
	// FSIPath is this dataset's link to the local filesystem if one exists
	FSIPath string `json:"fsiPath,omitempty"`
	//
	// Run Fields
	//
	// RunID is derived from from either the Commit.RunID, field or the runID of a
	// failed run. In the latter case the Path value will be empty
	RunID string `json:"runID,omitempty"`
	// RunStatus is a string version of the run.Status enumeration. This value
	// will always be one of:
	//    ""|"waiting"|"running"|"succeeded"|"failed"|"unchanged"|"skipped"
	// RunStatus is not stored on a dataset version, and instead must come from
	// either run state or a cache of run state
	// it's of type string to follow the "plain old data" pattern
	RunStatus string `json:"runStatus,omitempty"`
	// RunDuration is how long the run took/has currently taken in nanoseconds
	// default value of 0 means no duration data is available.
	// RunDuration is not stored on a dataset version, and instead must come from
	// either run state or a cache of run state
	RunDuration int64 `json:"runDuration,omitempty"`
}

// NewVersionInfoFromRef creates a sparse-populated VersionInfo from a dsref.Ref
func NewVersionInfoFromRef(ref Ref) VersionInfo {
	return VersionInfo{
		InitID:    ref.InitID,
		Username:  ref.Username,
		ProfileID: ref.ProfileID,
		Name:      ref.Name,
		Path:      ref.Path,
	}
}

// SimpleRef returns a simple dsref.Ref
func (v VersionInfo) SimpleRef() Ref {
	return Ref{
		InitID:    v.InitID,
		Username:  v.Username,
		ProfileID: v.ProfileID,
		Name:      v.Name,
		Path:      v.Path,
	}
}

// Alias returns the alias components of a Ref as a string
func (v *VersionInfo) Alias() string {
	s := v.Username
	if v.Name != "" {
		s += "/" + v.Name
	}
	return s
}

// ConvertDatasetToVersionInfo assigns values form a dataset to a VersionInfo
// This function is a shim while we work on building up dscache as a store of
// VersionInfo.
//
// Deprecated: Don't use this function for new code. Instead reference a
// VersionInfo that is stored somewhere, or write a function that builds a
// VersionInfo without needing a dataset
func ConvertDatasetToVersionInfo(ds *dataset.Dataset) VersionInfo {
	vi := VersionInfo{
		Username:  ds.Peername,
		ProfileID: ds.ProfileID,
		Name:      ds.Name,
		Path:      ds.Path,
	}
	if ds.Commit != nil {
		vi.CommitTime = ds.Commit.Timestamp
		vi.CommitTitle = ds.Commit.Title
		vi.CommitMessage = ds.Commit.Message
		vi.RunID = ds.Commit.RunID
	}
	if ds.Meta != nil {
		vi.MetaTitle = ds.Meta.Title
		if ds.Meta.Theme != nil {
			vi.ThemeList = strings.Join(ds.Meta.Theme, ",")
		}
	}

	if ds.Structure != nil {
		vi.BodyFormat = ds.Structure.Format
		vi.BodySize = ds.Structure.Length
		vi.BodyRows = ds.Structure.Entries
		vi.NumErrors = ds.Structure.ErrCount
	}

	return vi
}

// ConvertVersionInfoToDataset builds up a dataset from all the relevant
// VersionInfo fields.
//
// Deprecated: This function is needed only for supporting Search functionality.
// Do not add new callers if possible.
func ConvertVersionInfoToDataset(info *VersionInfo) *dataset.Dataset {
	return &dataset.Dataset{
		Peername:  info.Username,
		ProfileID: info.ProfileID,
		Name:      info.Name,
		Path:      info.Path,
		Commit: &dataset.Commit{
			Timestamp: info.CommitTime,
			Title:     info.CommitTitle,
			Message:   info.CommitMessage,
			RunID:     info.RunID,
		},
		Meta: &dataset.Meta{
			Title: info.MetaTitle,
		},
		Structure: &dataset.Structure{
			Format:   info.BodyFormat,
			Length:   info.BodySize,
			Entries:  info.BodyRows,
			ErrCount: info.NumErrors,
		},
	}
}

type lessFunc func(a, b *VersionInfo) bool

func newLessFunc(name string) (lessFunc, error) {
	switch name {
	case "name":
		return func(a, b *VersionInfo) bool { return (a.Username < b.Username && a.Name < b.Name) }, nil
	case "size":
		return func(a, b *VersionInfo) bool { return a.BodySize < b.BodySize }, nil
	}

	return nil, fmt.Errorf("unrecognized sorting field: %q", name)
}

// VersionInfoAggregator sorts slices of VersionInfos according to a provided
// string configuration. Call its Sort method to sort the data
// TODO(b5): add support for filtering
type VersionInfoAggregator struct {
	infos []VersionInfo
	less  []lessFunc
}

// Sort sorts the argument slice according to the less functions passed to OrderedBy.
func (agg *VersionInfoAggregator) Sort(infos []VersionInfo) {
	agg.infos = infos
	sort.Sort(agg)
}

// NewVersionInfoAggregator returns a Sorter that sorts using the less functions
// in order.
func NewVersionInfoAggregator(orderBy []string) (*VersionInfoAggregator, error) {
	less := []lessFunc{}
	for _, o := range orderBy {
		fn, err := newLessFunc(o)
		if err != nil {
			return nil, err
		}
		less = append(less, fn)
	}

	return &VersionInfoAggregator{
		less: less,
	}, nil
}

// Len is part of sort.Interface.
func (agg *VersionInfoAggregator) Len() int {
	return len(agg.infos)
}

// Swap is part of sort.Interface.
func (agg *VersionInfoAggregator) Swap(i, j int) {
	agg.infos[i], agg.infos[j] = agg.infos[j], agg.infos[i]
}

func (agg *VersionInfoAggregator) Less(i, j int) bool {
	p, q := &agg.infos[i], &agg.infos[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(agg.less)-1; k++ {
		less := agg.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return agg.less[k](p, q)
}
