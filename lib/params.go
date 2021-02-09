package lib

import (
	"fmt"
	"io"
	"net/http"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// DefaultPageSize is the max number of items in a page if no
// Limit param is provided to a paginated method
const DefaultPageSize = 100

// NZDefaultSetter modifies zero values to non-zero defaults when called
type NZDefaultSetter interface {
	SetNonZeroDefaults()
}

// RequestUnmarshaller is an interface for deserializing from an HTTP request
type RequestUnmarshaller interface {
	UnmarshalFromRequest(r *http.Request) error
}

// ListParams is the general input for any sort of Paginated Request
// ListParams define limits & offsets, not pages & page sizes.
// TODO - rename this to PageParams.
type ListParams struct {
	ProfileID profile.ID `json:"-"`
	Term      string
	Peername  string
	OrderBy   string
	Limit     int
	Offset    int
	// RPC is a horrible hack while we work to replace the net/rpc package
	// TODO - remove this
	RPC bool
	// Public only applies to listing datasets, shows only datasets that are
	// set to visible
	Public bool
	// ShowNumVersions only applies to listing datasets
	ShowNumVersions bool
	// EnsureFSIExists controls whether to ensure references in the repo have correct FSIPaths
	EnsureFSIExists bool
	// UseDscache controls whether to build a dscache to use to list the references
	UseDscache bool

	// Proxy identifies whether a call has been proxied from another instance
	Proxy bool
}

// SetNonZeroDefaults sets OrderBy to "created" if it's value is the empty string
func (p *ListParams) SetNonZeroDefaults() {
	if p.OrderBy == "" {
		p.OrderBy = "created"
	}
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *ListParams) UnmarshalFromRequest(r *http.Request) error {
	lp := ListParamsFromRequest(r)
	*p = lp
	return nil
}

// NewListParams creates a ListParams from page & pagesize, pages are 1-indexed
// (the first element is 1, not 0), NewListParams performs the conversion
func NewListParams(orderBy string, page, pageSize int) ListParams {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	return ListParams{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
}

// ListParamsFromRequest extracts ListParams from an http.Request pointer
func ListParamsFromRequest(r *http.Request) ListParams {
	var page, pageSize int
	if i := util.ReqParamInt(r, "page", 0); i != 0 {
		page = i
	}
	if i := util.ReqParamInt(r, "pageSize", 0); i != 0 {
		pageSize = i
	}
	return NewListParams(r.FormValue("orderBy"), page, pageSize)
}

// Page converts a ListParams struct to a util.Page struct
func (p ListParams) Page() util.Page {
	var number, size int
	size = p.Limit
	if size <= 0 {
		size = DefaultPageSize
	}
	number = p.Offset/size + 1
	return util.NewPage(number, size)
}

// SaveParams encapsulates arguments to Save
type SaveParams struct {
	// dataset supplies params directly, all other param fields override values
	// supplied by dataset
	Dataset *dataset.Dataset

	// dataset reference string, the name to save to
	Ref string
	// commit title, defaults to a generated string based on diff
	Title string
	// commit message, defaults to blank
	Message string
	// path to body data
	BodyPath string
	// absolute path or URL to the list of dataset files or components to load
	FilePaths []string
	// secrets for transform execution
	Secrets map[string]string
	// optional writer to have transform script record standard output to
	// note: this won't work over RPC, only on local calls
	ScriptOutput io.Writer `json:"-"`

	// TODO(dustmop): add `Wait bool`, if false, run the save asynchronously
	// and return events on the bus that provide the progress of the save operation

	// Apply runs a transform script to create the next version to save
	Apply bool
	// Replace writes the entire given dataset as a new snapshot instead of
	// applying save params as augmentations to the existing history
	Replace bool
	// option to make dataset private. private data is not currently implimented,
	// see https://github.com/qri-io/qri/issues/291 for updates
	Private bool
	// if true, convert body to the format of the previous version, if applicable
	ConvertFormatToPrev bool
	// comma separated list of component names to delete before saving
	Drop string
	// force a new commit, even if no changes are detected
	Force bool
	// save a rendered version of the template along with the dataset
	ShouldRender bool
	// new dataset only, don't create a commit on an existing dataset, name will be unused
	NewName bool
	// whether to create a new dscache if none exists
	UseDscache bool

	// Proxy identifies whether a call has been proxied from another instance
	Proxy bool
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *SaveParams) UnmarshalFromRequest(r *http.Request) error {
	if p.Dataset == nil {
		return fmt.Errorf("dataset missing")
	}
	ref := reporef.DatasetRef{
		Name:     p.Dataset.Name,
		Peername: p.Dataset.Peername,
	}
	p.Ref = ref.AliasString()
	return nil
}

// AbsolutizePaths converts any relative path references to their absolute
// variations, safe to call on a nil instance
func (p *SaveParams) AbsolutizePaths() error {
	if p == nil {
		return nil
	}

	for i := range p.FilePaths {
		if err := qfs.AbsPath(&p.FilePaths[i]); err != nil {
			return err
		}
	}

	if err := qfs.AbsPath(&p.BodyPath); err != nil {
		return fmt.Errorf("body file: %w", err)
	}
	return nil
}

// GetParams defines parameters for looking up the head or body of a dataset
type GetParams struct {
	// Refstr to get, representing a dataset ref to be parsed
	Refstr   string
	Selector string

	// read from a filesystem link instead of stored version
	Format       string
	FormatConfig dataset.FormatConfig

	Limit, Offset int
	All           bool

	// outfile is a filename to save the dataset to
	Outfile string
	// whether to generate a filename from the dataset name instead
	GenFilename bool
	Remote      string

	// Proxy identifies whether a call has been proxied from another instance
	Proxy bool
}

// GetResult combines data with it's hashed path
type GetResult struct {
	Ref       *dsref.Ref       `json:"ref"`
	Dataset   *dataset.Dataset `json:"data"`
	Bytes     []byte           `json:"bytes"`
	Message   string           `json:"message"`
	FSIPath   string           `json:"fsipath"`
	Published bool             `json:"published"`
}

// RenameParams defines parameters for Dataset renaming
type RenameParams struct {
	Current, Next string
}

// RemoveParams defines parameters for remove command
type RemoveParams struct {
	Ref       string
	Revision  dsref.Rev
	KeepFiles bool
	Force     bool
	Proxy     bool
}

// RemoveResponse gives the results of a remove
type RemoveResponse struct {
	Ref        string
	NumDeleted int
	Message    string
	Unlinked   bool
}

// PullParams encapsulates parameters to the add command
type PullParams struct {
	Ref      string
	LinkDir  string
	Remote   string // remote to attempt to pull from
	LogsOnly bool   // only fetch logbook data
}
