package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dag"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/automation/run"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/base/fill"
	"github.com/qri-io/qri/dsref"
	qrierr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/transform"
)

// DatasetMethods work with datasets, creating new versions (save), reading
// dataset data (get), deleting versions (remove), and moving datasets over
// network connections (push & pull)
type DatasetMethods struct {
	d dispatcher
}

// Name returns the name of this method group
func (m DatasetMethods) Name() string {
	return "dataset"
}

// Attributes defines attributes for each method
func (m DatasetMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"componentstatus": {Endpoint: AEComponentStatus, HTTPVerb: "POST"},
		"get":             {Endpoint: AEGet, HTTPVerb: "POST"},
		"getcsv":          {Endpoint: DenyHTTP}, // getcsv is not part of the json api, but is handled in a separate `GetBodyCSVHandler` function
		"getzip":          {Endpoint: DenyHTTP}, // getzip is not part of the json api, but is handled is a separate `GetHandler` function
		"activity":        {Endpoint: AEActivity, HTTPVerb: "POST"},
		"rename":          {Endpoint: AERename, HTTPVerb: "POST", DefaultSource: "local"},
		"save":            {Endpoint: AESave, HTTPVerb: "POST"},
		"pull":            {Endpoint: AEPull, HTTPVerb: "POST", DefaultSource: "network"},
		"push":            {Endpoint: AEPush, HTTPVerb: "POST", DefaultSource: "local"},
		"render":          {Endpoint: AERender, HTTPVerb: "POST"},
		"remove":          {Endpoint: AERemove, HTTPVerb: "POST", DefaultSource: "local"},
		"validate":        {Endpoint: AEValidate, HTTPVerb: "POST", DefaultSource: "local"},
		"manifest":        {Endpoint: AEManifest, HTTPVerb: "POST", DefaultSource: "local"},
		"manifestmissing": {Endpoint: AEManifestMissing, HTTPVerb: "POST", DefaultSource: "local"},
		"daginfo":         {Endpoint: AEDAGInfo, HTTPVerb: "POST", DefaultSource: "local"},
	}
}

// GetParams defines parameters for looking up the head or body of a dataset
type GetParams struct {
	// dataset reference to fetch; e.g. "b5/world_bank_population"
	Ref string `json:"ref"`
	// a component or nested field names to extract from the dataset; e.g. "body"
	Selector string `json:"selector"`
	// number of results to limit to. only applies when selector is "body"
	Limit int `json:"limit"`
	// number of results to skip. only applies when selector is "body"
	Offset int `json:"offset"`
	// TODO(dustmop): Remove `All` once `Cursor` is in use. Instead, callers should
	// loop over their `Cursor` in order to get all rows.
	All bool `json:"all" docs:"hidden"`
}

// SetNonZeroDefaults assigns default values
func (p *GetParams) SetNonZeroDefaults() {
	if p.Selector == "body" {
		if !p.All {
			// ensure valid limit value
			if p.Limit <= 0 {
				p.Limit = 25
			}
			// ensure valid offset value
			if p.Offset < 0 {
				p.Offset = 0
			}

		}
	}
}

// Validate returns an error if GetParams fields are in an invalid state
func (p *GetParams) Validate() error {
	if !isValidSelector(p.Selector) {
		return fmt.Errorf("could not parse request: invalid selector")
	}
	if p.Selector == "body" {
		if !p.All && (p.Limit < 0 || p.Offset < 0) {
			return fmt.Errorf("invalid limit / offset settings")
		}
	}

	return nil
}

func isValidSelector(selector string) bool {
	return validSelector.MatchString(selector)
}

var validSelector = regexp.MustCompile(`^$|^[\w-\.]*[\w]$`)

// UnmarshalFromRequest satisfies the Unmarshaller interface
func (p *GetParams) UnmarshalFromRequest(r *http.Request) error {
	log.Debugf("GetParams.UnmarshalFromRequest ref:%s", r.FormValue("ref"))
	p.Ref = r.FormValue("ref")

	ref, err := dsref.Parse(p.Ref)
	if err != nil {
		return err
	}

	if ref.Username == "me" {
		return fmt.Errorf("username \"me\" not allowed")
	}

	p.Selector = r.FormValue("selector")

	p.All = util.ReqParamBool(r, "all", true)
	p.Limit = util.ReqParamInt(r, "limit", 0)
	p.Offset = util.ReqParamInt(r, "offset", 0)
	if !(p.Offset == 0 && p.Limit == 0) {
		p.All = false
	}
	return nil
}

// GetResult returns the dataset or some part of it as structured data inside the `Value` field
// The `Bytes` field is reserved for data that can only be expressed as a slice of bytes
// Byte slices must be treated as a special case because of json.Marshal. json.Marshal will serialize
// a slice of bytes as base64 encoded json. If this is deserialized into an `interface{}`, it will
// remain a string. It needs to be explicitly deserialized into a `[]byte` field to not degredate the information
type GetResult struct {
	Value interface{} `json:"value,omitempty"`
	Bytes []byte      `json:"bytes,omitempty"`
}

// DataResponse is the struct used to respond to api requests made to the /body endpoint
// It is necessary because we need to include the 'path' field in the response
type DataResponse struct {
	Path string          `json:"path"`
	Data json.RawMessage `json:"data"`
}

// Get retrieves datasets and components for a given reference. p.Ref is parsed to create
// a reference, which is used to load the dataset. It will be loaded from the local repo
// or from the filesystem if it has a linked working directory.
// Using p.Selector will control what components are returned in res.Value. The default,
// a blank selector, will also fill the entire dataset at res.Value. If the selector contains ".script"
// then res.Bytes is loaded with the script contents as bytes. If the selector is "stats", then res.Value is loaded
// with the generated stats.
func (m DatasetMethods) Get(ctx context.Context, p *GetParams) (*GetResult, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "get"), p)
	if res, ok := got.(*GetResult); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// GetCSV fetches the body as a csv encoded byte slice, it recognizes Limit, Offset, and All list params
func (m DatasetMethods) GetCSV(ctx context.Context, p *GetParams) ([]byte, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "getcsv"), p)
	if res, ok := got.([]byte); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// GetZipResults is returned by `GetZip`
// It contains a byte slice of the compressed data as well as a generated name based on the dataset
type GetZipResults struct {
	Bytes         []byte
	GeneratedName string
}

// GetZip fetches an entire dataset as a zip archive
func (m DatasetMethods) GetZip(ctx context.Context, p *GetParams) (*GetZipResults, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "getzip"), p)
	if res, ok := got.(*GetZipResults); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

func scriptFileSelection(ds *dataset.Dataset, selector string) (qfs.File, bool) {
	parts := strings.Split(selector, ".")
	if len(parts) != 2 {
		return nil, false
	}
	if parts[1] != "script" {
		return nil, false
	}
	if parts[0] == "transform" && ds.Transform != nil && ds.Transform.ScriptFile() != nil {
		return ds.Transform.ScriptFile(), true
	} else if parts[0] == "readme" && ds.Readme != nil && ds.Readme.ScriptFile() != nil {
		return ds.Readme.ScriptFile(), true
	} else if parts[0] == "viz" && ds.Viz != nil && ds.Viz.ScriptFile() != nil {
		return ds.Viz.ScriptFile(), true
	} else if parts[0] == "rendered" && ds.Viz != nil && ds.Viz.RenderedFile() != nil {
		return ds.Viz.RenderedFile(), true
	}
	return nil, false
}

// ActivityParams defines parameters for the Activity method
type ActivityParams struct {
	ListParams
	// Reference to data to fetch history for; e.g. "b5/world_bank_population"
	Ref string `json:"ref"`
	// if true, pull any datasets that aren't stored locally; e.g. false
	Pull bool `json:"pull"`
}

// Activity returns the activity and changes for a given dataset
func (m DatasetMethods) Activity(ctx context.Context, params *ActivityParams) ([]dsref.VersionInfo, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "activity"), params)
	if res, ok := got.([]dsref.VersionInfo); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// SaveParams encapsulates arguments to Save
type SaveParams struct {
	// dataset supplies params directly, all other param fields override values
	// supplied by dataset
	Dataset *dataset.Dataset

	// dataset reference string, the name to save to; e.g. "b5/world_bank_population"
	Ref string `json:"ref"`
	// commit title, defaults to a generated string based on diff; e.g. "update dataset meta"
	Title string `json:"title"`
	// commit message, defaults to blank; e.g. "reaname title & fill in supported langages"
	Message string
	// path to body data
	BodyPath string `json:"bodyPath" qri:"fspath"`
	// absolute path or URL to the list of dataset files or components to load
	FilePaths []string `json:"filePaths" qri:"fspath"`
	// secrets for transform execution. Should be a set of key: value pairs
	Secrets map[string]string `json:"secrets"`
	// optional writer to have transform script record standard output to
	// note: this won't work over RPC, only on local calls
	ScriptOutput io.Writer `json:"-"`

	// TODO(dustmop): add `Wait bool`, if false, run the save asynchronously
	// and return events on the bus that provide the progress of the save operation

	// Apply runs a transform script to create the next version to save
	Apply bool `json:"apply"`
	// Replace writes the entire given dataset as a new snapshot instead of
	// applying save params as augmentations to the existing history
	Replace bool `json:"replace"`
	// option to make dataset private. private data is not currently implimented,
	// see https://github.com/qri-io/qri/issues/291 for updates
	Private bool `json:"private"`
	// if true, convert body to the format of the previous version, if applicable
	ConvertFormatToPrev bool `json:"convertFormatToPrev"`
	// comma separated list of component names to delete before saving
	Drop string `json:"drop"`
	// force a new commit, even if no changes are detected
	Force bool `json:"force"`
	// save a rendered version of the template along with the dataset
	ShouldRender bool `json:"shouldRender"`
	// new dataset only, don't create a commit on an existing dataset, name will be unused
	NewName bool `json:"newName"`
}

// SetNonZeroDefaults sets basic save path params to defaults
func (p *SaveParams) SetNonZeroDefaults() {
	p.ConvertFormatToPrev = true
}

// Save adds a history entry, updating a dataset
func (m DatasetMethods) Save(ctx context.Context, p *SaveParams) (*dataset.Dataset, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "save"), p)
	if res, ok := got.(*dataset.Dataset); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// RenameParams defines parameters for Dataset renaming
type RenameParams struct {
	Current string `json:"current"`
	Next    string `json:"next"`
}

// Rename changes a user's given name for a dataset
func (m DatasetMethods) Rename(ctx context.Context, p *RenameParams) (*dsref.VersionInfo, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "rename"), p)
	if res, ok := got.(*dsref.VersionInfo); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// RemoveParams defines parameters for remove command
type RemoveParams struct {
	Ref       string     `json:"ref"`
	Revision  *dsref.Rev `json:"revision"`
	KeepFiles bool       `json:"keepFiles"`
	Force     bool       `json:"force"`
}

// RemoveResponse gives the results of a remove
type RemoveResponse struct {
	Ref        string `json:"ref"`
	NumDeleted int    `json:"numDeleted"`
	Message    string `json:"message"`
	Unlinked   bool   `json:"unlinked"`
}

// SetNonZeroDefaults assigns default values
func (p *RemoveParams) SetNonZeroDefaults() {
	if p.Revision == nil {
		p.Revision = &dsref.Rev{Field: "ds", Gen: -1}
	}
}

// ErrCantRemoveDirectoryDirty is returned when a directory is dirty so the files cant' be removed
var ErrCantRemoveDirectoryDirty = fmt.Errorf("cannot remove files while working directory is dirty")

// Remove a dataset entirely or remove a certain number of revisions
func (m DatasetMethods) Remove(ctx context.Context, p *RemoveParams) (*RemoveResponse, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "remove"), p)
	if res, ok := got.(*RemoveResponse); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// PullParams encapsulates parameters to the add command
type PullParams struct {
	Ref     string `json:"ref"`
	LinkDir string `json:"linkDir" qri:"fspath"`
	// only fetch logbook data
	LogsOnly bool `json:"logsOnly"`
}

// Pull downloads and stores an existing dataset to a peer's repository via
// a network connection
func (m DatasetMethods) Pull(ctx context.Context, p *PullParams) (*dataset.Dataset, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "pull"), p)
	if res, ok := got.(*dataset.Dataset); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// PushParams encapsulates parmeters for dataset publication
type PushParams struct {
	Ref    string `json:"ref" schema:"ref"`
	Remote string `json:"remote"`
	// All indicates all versions of a dataset and the dataset namespace should
	// be either published or removed
	All bool `json:"all"`
}

// Push posts a dataset version to a remote
func (m DatasetMethods) Push(ctx context.Context, p *PushParams) (*dsref.Ref, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "push"), p)
	if res, ok := got.(*dsref.Ref); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// ValidateParams defines parameters for dataset data validation
type ValidateParams struct {
	Ref               string `json:"ref"`
	BodyFilename      string `json:"bodyFilename" qri:"fspath"`
	SchemaFilename    string `json:"schemaFilename" qri:"fspath"`
	StructureFilename string `json:"structureFilename" qri:"fspath"`
}

// ValidateResponse is the result of running validate against a dataset
type ValidateResponse struct {
	// Structure used to perform validation
	Structure *dataset.Structure `json:"structure"`
	// Validation Errors
	Errors []jsonschema.KeyError `json:"errors"`
}

// Validate gives a dataset of errors and issues for a given dataset
func (m DatasetMethods) Validate(ctx context.Context, p *ValidateParams) (*ValidateResponse, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "validate"), p)
	if res, ok := got.(*ValidateResponse); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// ManifestParams encapsulates parameters to the manifest command
type ManifestParams struct {
	Ref string `json:"ref"`
}

// Manifest generates a manifest for a dataset path
func (m DatasetMethods) Manifest(ctx context.Context, p *ManifestParams) (*dag.Manifest, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "manifest"), p)
	if res, ok := got.(*dag.Manifest); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// ManifestMissingParams encapsulates parameters to the missing manifest command
type ManifestMissingParams struct {
	Manifest *dag.Manifest `json:"manifest"`
}

// ManifestMissing generates a manifest of blocks that are not present on this repo for a given manifest
func (m DatasetMethods) ManifestMissing(ctx context.Context, p *ManifestMissingParams) (*dag.Manifest, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "manifestmissing"), p)
	if res, ok := got.(*dag.Manifest); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// DAGInfoParams defines parameters for the DAGInfo method
type DAGInfoParams struct {
	Ref   string `json:"ref"`
	Label string `json:"label"`
}

// DAGInfo generates a dag.Info for a dataset path. If a label is given, DAGInfo will generate a sub-dag.Info at that label.
func (m DatasetMethods) DAGInfo(ctx context.Context, p *DAGInfoParams) (*dag.Info, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "daginfo"), p)
	if res, ok := got.(*dag.Info); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// ComponentStatus gets changes that happened at a particular version in the history of the given
// dataset reference.
func (m DatasetMethods) ComponentStatus(ctx context.Context, p *LinkParams) ([]StatusItem, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "componentstatus"), p)
	if res, ok := got.([]StatusItem); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// formFileDataset extracts a dataset document from a http Request
func formFileDataset(r *http.Request, ds *dataset.Dataset) (err error) {
	datafile, dataHeader, err := r.FormFile("file")
	if err == http.ErrMissingFile {
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening dataset file: %s", err)
		return
	}
	if datafile != nil {
		switch strings.ToLower(filepath.Ext(dataHeader.Filename)) {
		case ".yaml", ".yml":
			var data []byte
			data, err = ioutil.ReadAll(datafile)
			if err != nil {
				err = fmt.Errorf("reading dataset file: %w", err)
				return
			}
			fields := &map[string]interface{}{}
			if err = yaml.Unmarshal(data, fields); err != nil {
				err = fmt.Errorf("deserializing YAML file: %w", err)
				return
			}
			if err = fill.Struct(*fields, ds); err != nil {
				return
			}
		case ".json":
			if err = json.NewDecoder(datafile).Decode(ds); err != nil {
				err = fmt.Errorf("error decoding json file: %s", err)
				return
			}
		}
	}

	if peername := r.FormValue("peername"); peername != "" {
		ds.Peername = peername
	}
	if name := r.FormValue("name"); name != "" {
		ds.Name = name
	}
	if bp := r.FormValue("body_path"); bp != "" {
		ds.BodyPath = bp
	}

	tfFile, tfHeader, err := r.FormFile("transform")
	if err == http.ErrMissingFile {
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening transform file: %s", err)
		return
	}
	if tfFile != nil {
		var tfData []byte
		if tfData, err = ioutil.ReadAll(tfFile); err != nil {
			return
		}
		if ds.Transform == nil {
			ds.Transform = &dataset.Transform{}
		}
		ds.Transform.Syntax = "starlark"
		ds.Transform.ScriptBytes = tfData
		ds.Transform.ScriptPath = tfHeader.Filename
	}

	vizFile, vizHeader, err := r.FormFile("viz")
	if err == http.ErrMissingFile {
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening viz file: %s", err)
		return
	}
	if vizFile != nil {
		var vizData []byte
		if vizData, err = ioutil.ReadAll(vizFile); err != nil {
			return
		}
		if ds.Viz == nil {
			ds.Viz = &dataset.Viz{}
		}
		ds.Viz.Format = "html"
		ds.Viz.ScriptBytes = vizData
		ds.Viz.ScriptPath = vizHeader.Filename
	}

	bodyfile, bodyHeader, err := r.FormFile("body")
	if err == http.ErrMissingFile {
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening body file: %s", err)
		return
	}
	if bodyfile != nil {
		var bodyData []byte
		if bodyData, err = ioutil.ReadAll(bodyfile); err != nil {
			return
		}
		ds.BodyPath = bodyHeader.Filename
		ds.BodyBytes = bodyData

		if ds.Structure == nil {
			// TODO - this is silly and should move into base.PrepareDataset funcs
			ds.Structure = &dataset.Structure{}
			format, comp, err := detect.FormatFromFilename(bodyHeader.Filename)
			if err != nil {
				return err
			}
			st, _, err := detect.FromReader(format, comp, bytes.NewReader(ds.BodyBytes))
			if err != nil {
				return err
			}
			ds.Structure = st
		}
	}

	return
}

// RenderParams defines parameters for the Render method
type RenderParams struct {
	// Ref is a string reference to the dataset to render
	Ref string `json:"ref"`
	// Optionally pass an entire dataset in for rendering, if providing a dataset,
	// the Ref field must be empty
	Dataset *dataset.Dataset `json:"dataset"`
	// Optional template override
	Template []byte `json:"template"`
	// TODO (b5): investigate if this field is still in use
	UseFSI bool `json:"useFSI"`
	// Output format. defaults to "html"
	Format string `json:"format"`
	// Selector
	Selector string `json:"selector"`
}

// SetNonZeroDefaults assigns default values
func (p *RenderParams) SetNonZeroDefaults() {
	if p.Format == "" {
		p.Format = "html"
	}
}

// Validate checks if render parameters are valid
func (p *RenderParams) Validate() error {
	if p.Ref != "" && p.Dataset != nil {
		return fmt.Errorf("cannot provide both a reference and a dataset to render")
	}
	if p.Ref == "" && p.Dataset == nil {
		return dsref.ErrEmptyRef
	}
	if p.Selector == "" {
		return fmt.Errorf("selector must be one of 'viz' or 'readme'")
	}
	return nil
}

// Render renders a viz or readme component as html
func (m DatasetMethods) Render(ctx context.Context, p *RenderParams) ([]byte, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "render"), p)
	if res, ok := got.([]byte); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// datasetImpl holds the method implementations for DatasetMethods
type datasetImpl struct{}

// Get retrieves datasets and components for a given reference.t
func (datasetImpl) Get(scope scope, p *GetParams) (*GetResult, error) {
	ref, ds, err := openAndLoadFSIEnabledDataset(scope, p)
	if err != nil {
		return nil, err
	}

	res := &GetResult{}

	if p.Selector == "body" {
		// `qri get body` loads the body
		if !p.All && (p.Limit < 0 || p.Offset < 0) {
			return nil, fmt.Errorf("invalid limit / offset settings")
		}
		if fsi.IsFSIPath(ref.Path) {
			// TODO(dustmop): Need to handle the special case where an FSI directory has a body
			// but no structure, which should infer a schema in order to read the body. Once that
			// works we can remove the fsi.GetBody call and just use base.GetBody.
			fsiPath := fsi.FilesystemPathToLocal(ref.Path)
			res.Value, err = fsi.GetBody(fsiPath, p.Offset, p.Limit, p.All)
			if err != nil {
				log.Debugf("Get dataset, fsi.GetBody %q failed, error: %s", fsiPath, err)
				return nil, err
			}
			return res, nil
		}
		res.Value, err = base.GetBody(ds, p.Limit, p.Offset, p.All)
		if err != nil {
			log.Debugf("Get dataset, base.GetBody %q failed, error: %s", ds, err)
			return nil, err
		}
		return res, nil
	} else if scriptFile, ok := scriptFileSelection(ds, p.Selector); ok {
		// Fields that have qfs.File types should be read and returned
		res.Bytes, err = ioutil.ReadAll(scriptFile)
		if err != nil {
			return nil, err
		}
		return res, nil
	} else if p.Selector == "stats" {
		sa, err := scope.Stats().Stats(scope.Context(), ds)
		if err != nil {
			return nil, err
		}
		res.Value = sa.Stats
		return res, nil
	}

	// `qri get <selector>` loads only the applicable component / field
	res.Value, err = base.ApplyPath(ds, p.Selector)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func openAndLoadFSIEnabledDataset(scope scope, p *GetParams) (*dsref.Ref, *dataset.Dataset, error) {
	scope.EnableWorkingDir(true)
	ds, err := scope.Loader().LoadDataset(scope.Context(), p.Ref)
	if err != nil {
		return nil, nil, err
	}

	ref := dsref.ConvertDatasetToVersionInfo(ds).SimpleRef()

	if fsi.IsFSIPath(ref.Path) {
		ds.Path = ""
	}

	if err = base.OpenDataset(scope.Context(), scope.Filesystem(), ds); err != nil {
		log.Debugf("base.OpenDataset failed, error: %s", err)
		return nil, nil, err
	}
	return &ref, ds, nil
}

func (datasetImpl) GetCSV(scope scope, p *GetParams) ([]byte, error) {
	return getBodyBytes(scope, p, dataset.CSVDataFormat, nil)
}

func getBodyBytes(scope scope, p *GetParams, format dataset.DataFormat, fc dataset.FormatConfig) ([]byte, error) {
	ref, ds, err := openAndLoadFSIEnabledDataset(scope, p)
	if err != nil {
		return nil, err
	}

	if fc == nil && ds.Structure != nil {
		// if we aren't given any format configuration &
		// the format given matches the format of the body, pull in any known format configuration
		if ds.Structure.DataFormat() == format {
			fc, err = dataset.ParseFormatConfigMap(format, ds.Structure.FormatConfig)
			if err != nil {
				log.Debugf("getBodyBytes: dataset.ParseFormatConfigMap failed: %s", err)
				return nil, err
			}
		}
	}

	if fsi.IsFSIPath(ref.Path) {
		// TODO(dustmop): Need to handle the special case where an FSI directory has a body
		// but no structure, which should infer a schema in order to read the body. Once that
		// works we can remove the fsi.ReadBodyBytes call and just use base.ReadBodyBytes
		fsiPath := fsi.FilesystemPathToLocal(ref.Path)
		bodyBytes, err := fsi.ReadBodyBytes(fsiPath, dataset.CSVDataFormat, fc, p.Offset, p.Limit, p.All)
		if err != nil {
			log.Debugf("lib.getBodyBytes, fsi.GetBody %q failed, error: %s", fsiPath, err)
			return nil, err
		}
		return bodyBytes, nil
	}
	bodyBytes, err := base.ReadBodyBytes(ds, dataset.CSVDataFormat, fc, p.Limit, p.Offset, p.All)
	if err != nil {
		log.Debugf("lib.getBodyBytes, body, base.GetBody %q failed, error: %s", ds, err)
		return nil, err
	}
	return bodyBytes, nil
}

func (datasetImpl) GetZip(scope scope, p *GetParams) (*GetZipResults, error) {
	ref, ds, err := openAndLoadFSIEnabledDataset(scope, p)
	if err != nil {
		return nil, err
	}

	var outBuf bytes.Buffer
	var zipFile io.Writer
	zipFile = &outBuf
	// TODO(dustmop): This function is inefficient and a poor use of logbook, but it's
	// necessary until dscache is in use.
	initID, err := scope.Logbook().RefToInitID(*ref)
	if err != nil {
		return nil, err
	}
	err = archive.WriteZip(scope.Context(), scope.Filesystem(), ds, "json", initID, *ref, zipFile)
	if err != nil {
		return nil, err
	}
	filename, err := archive.GenerateFilename(ds, "zip")
	if err != nil {
		return nil, err
	}
	return &GetZipResults{Bytes: outBuf.Bytes(), GeneratedName: filename}, nil
}

// Activity returns the activity and changes for a given dataset
func (datasetImpl) Activity(scope scope, params *ActivityParams) ([]dsref.VersionInfo, error) {
	// ensure valid limit value
	if params.Limit <= 0 {
		params.Limit = 25
	}
	// ensure valid offset value
	if params.Offset < 0 {
		params.Offset = 0
	}

	if params.Pull && scope.SourceName() != "network" {
		return nil, fmt.Errorf("cannot pull without using network source")
	}

	ref, location, err := scope.ParseAndResolveRef(scope.Context(), params.Ref)
	if err != nil {
		return nil, err
	}

	if location == "" {
		// local resolution
		return base.DatasetLog(scope.Context(), scope.Repo(), ref, params.Limit, params.Offset, true)
	}

	logs, err := scope.RemoteClient().FetchLogs(scope.Context(), ref, location)
	if err != nil {
		return nil, err
	}

	// TODO (b5) - FetchLogs currently returns oplogs arranged in user > dataset > branch
	// hierarchy, and we need to descend to the branch oplog to get commit history
	// info. It might be nicer if FetchLogs instead returned the branch oplog, but
	// with .Parent() fields loaded & connected
	if len(logs.Logs) > 0 {
		logs = logs.Logs[0]
		if len(logs.Logs) > 0 {
			logs = logs.Logs[0]
		}
	}

	items := logbook.ConvertLogsToVersionInfos(logs, ref)
	log.Debugf("found %d items: %v", len(items), items)
	if len(items) == 0 {
		return nil, repo.ErrNoHistory
	}

	for i, item := range items {
		local, hasErr := scope.Filesystem().Has(scope.Context(), item.Path)
		if hasErr != nil {
			continue
		}
		items[i].Foreign = !local

		if local {
			if ds, err := dsfs.LoadDataset(scope.Context(), scope.Repo().Filesystem(), item.Path); err == nil {
				if ds.Commit != nil {
					items[i].CommitMessage = ds.Commit.Message
				}
			}
		}
	}

	return items, nil
}

// IsSelectorScriptFile takes a selector string and returns true if the selector contains "script"
func IsSelectorScriptFile(selector string) bool {
	if selector == "" {
		return false
	}
	parts := strings.Split(selector, ".")
	if len(parts) != 2 {
		return false
	}
	if parts[1] == "script" {
		return true
	}
	return false
}

// Save adds a history entry, updating a dataset
func (datasetImpl) Save(scope scope, p *SaveParams) (*dataset.Dataset, error) {
	log.Debugw("DatasetMethods.Save", "ref", p.Ref, "apply", p.Apply)
	res := &dataset.Dataset{}

	var (
		writeDest = scope.Filesystem().DefaultWriteFS() // filesystem dataset will be written to
		pro       = scope.Repo().Profiles().Owner()     // user making the request. hard-coded to repo owner
	)

	if p.Private {
		return nil, fmt.Errorf("option to make dataset private not yet implemented, refer to https://github.com/qri-io/qri/issues/291 for updates")
	}

	// If the dscache doesn't exist yet, it will only be created if the appropriate flag enables it.
	if scope.UseDscache() {
		c := scope.Dscache()
		c.CreateNewEnabled = true
	}

	// start with dataset fields provided by params
	ds := p.Dataset
	if ds == nil {
		ds = &dataset.Dataset{}
	}
	ds.Assign(&dataset.Dataset{
		BodyPath: p.BodyPath,
		Commit: &dataset.Commit{
			Title:   p.Title,
			Message: p.Message,
		},
	})

	if len(p.FilePaths) > 0 {
		// TODO (b5): handle this with a qfs.Filesystem
		dsf, err := ReadDatasetFiles(p.FilePaths...)
		if err != nil {
			return nil, err
		}
		dsf.Assign(ds)
		ds = dsf
	}

	if p.Ref == "" && ds.Name != "" {
		p.Ref = fmt.Sprintf("me/%s", ds.Name)
	}

	resolver, err := scope.LocalResolver()
	if err != nil {
		log.Debugw("save construct local resolver", "err", err)
		return nil, err
	}

	ref, isNew, err := base.PrepareSaveRef(scope.Context(), pro, scope.Logbook(), resolver, p.Ref, ds.BodyPath, p.NewName)
	if err != nil {
		log.Debugw("save PrepareSaveRef", "refParam", p.Ref, "wantNewName", p.NewName, "err", err)
		return nil, err
	}

	success := false
	defer func() {
		// if creating a new dataset fails, we need to remove the dataset
		if isNew && !success {
			log.Debugf("removing unused log for new dataset %s", ref)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			if err := scope.Logbook().RemoveLog(ctx, ref); err != nil {
				log.Errorf("couldn't cleanup unused reference: %q", err)
			}
			cancel()
		}
	}()

	ds.Name = ref.Name
	ds.Peername = ref.Username

	var fsiPath string
	if !isNew {
		// check for FSI linked data
		fsiRef := ref.Copy()
		if err := scope.FSISubsystem().ResolvedPath(&fsiRef); err == nil {
			fsiPath = fsi.FilesystemPathToLocal(fsiRef.Path)
			fsiDs, err := fsi.ReadDir(fsiPath)
			if err != nil {
				return nil, err
			}
			fsiDs.Assign(ds)
			ds = fsiDs
		}
	}

	if !p.Force &&
		!p.Apply &&
		p.Drop == "" &&
		ds.BodyPath == "" &&
		ds.Body == nil &&
		ds.BodyBytes == nil &&
		ds.Structure == nil &&
		ds.Meta == nil &&
		ds.Readme == nil &&
		ds.Viz == nil &&
		ds.Transform == nil {
		return nil, fmt.Errorf("no changes to save")
	}

	if err = base.OpenDataset(scope.Context(), scope.Filesystem(), ds); err != nil {
		log.Debugw("save OpenDataset", "err", err.Error())
		return nil, err
	}

	// runState holds the results of transform application. will be non-nil if a
	// transform is applied while saving
	var runState *run.State

	// If applying a transform, execute its script before saving
	if p.Apply {
		if ds.Transform == nil {
			// if no transform component exists, load the latest transform component
			// from history
			if isNew {
				return nil, fmt.Errorf("cannot apply while saving without a transform")
			}

			prevTransformDataset, err := base.LoadRevs(scope.Context(), scope.Filesystem(), ref, []*dsref.Rev{{Field: "tf", Gen: 1}})
			if err != nil {
				return nil, fmt.Errorf("loading transform component from history: %w", err)
			}
			ds.Transform = prevTransformDataset.Transform
		}

		scriptOut := p.ScriptOutput
		secrets := p.Secrets
		runID := ds.Commit.RunID
		if runID == "" {
			// if there is no given runID, allocate an ID for the transform,
			// subscribe to print output & build up the run.State
			runID = run.NewID()
		}
		runState = &run.State{ID: runID}

		scope.Bus().SubscribeID(func(ctx context.Context, e event.Event) error {
			runState.AddTransformEvent(e)
			if e.Type == event.ETTransformPrint {
				if msg, ok := e.Payload.(event.TransformMessage); ok {
					if p.ScriptOutput != nil {
						io.WriteString(scriptOut, msg.Msg)
						io.WriteString(scriptOut, "\n")
					}
				}
			}
			return nil
		}, runID)

		// apply the transform
		shouldWait := true
		transformer := transform.NewTransformer(scope.AppContext(), scope.Loader(), scope.Bus())
		if err := transformer.Apply(scope.Context(), ds, runID, shouldWait, scriptOut, secrets); err != nil {
			log.Errorw("transform run error", "err", err.Error())
			runState.Message = err.Error()
			if err := scope.Logbook().WriteTransformRun(scope.Context(), ref.InitID, runState); err != nil {
				log.Debugw("writing errored transform run to logbook:", "err", err.Error())
				return nil, err
			}

			return nil, err
		}

		ds.Commit.RunID = runID
	}

	if fsiPath != "" && p.Drop != "" {
		return nil, qrierr.New(fmt.Errorf("cannot drop while FSI-linked"), "can't drop component from a working-directory, delete files instead.")
	}

	fileHint := p.BodyPath
	if len(p.FilePaths) > 0 {
		fileHint = p.FilePaths[0]
	}

	switches := base.SaveSwitches{
		FileHint:            fileHint,
		Replace:             p.Replace,
		Pin:                 true,
		ConvertFormatToPrev: p.ConvertFormatToPrev,
		ForceIfNoChanges:    p.Force,
		ShouldRender:        p.ShouldRender,
		NewName:             p.NewName,
		Drop:                p.Drop,
	}
	savedDs, err := base.SaveDataset(scope.Context(), scope.Repo(), writeDest, ref.InitID, ref.Path, ds, runState, switches)
	if err != nil {
		// datasets that are unchanged & have a runState record a record of no-changes
		// to logbook
		if errors.Is(err, dsfs.ErrNoChanges) && runState != nil {
			runState.Status = run.RSUnchanged
			runState.Message = err.Error()
			if err := scope.Logbook().WriteTransformRun(scope.Context(), ref.InitID, runState); err != nil {
				log.Debugw("writing unchanged transform run to logbook:", "err", err.Error())
				return nil, err
			}
		}

		log.Debugw("save base.SaveDataset", "err", err)
		return nil, err
	}

	success = true
	*res = *savedDs

	// TODO (b5) - this should be integrated into base.SaveDataset
	if fsiPath != "" {
		vi := dsref.ConvertDatasetToVersionInfo(savedDs)
		vi.FSIPath = fsiPath
		if err = repo.PutVersionInfoShim(scope.Context(), scope.Repo(), &vi); err != nil {
			log.Debugw("save PutVersionInfoShim", "fsiPath", fsiPath, "err", err)
			return nil, err
		}
		// Need to pass filesystem here so that we can read the README component and write it
		// properly back to disk.
		if writeErr := fsi.WriteComponents(savedDs, fsiPath, scope.Filesystem()); err != nil {
			log.Error(writeErr)
		}
	}

	return res, nil
}

// Rename changes a user's given name for a dataset
func (datasetImpl) Rename(scope scope, p *RenameParams) (*dsref.VersionInfo, error) {
	if p.Current == "" {
		return nil, fmt.Errorf("current name is required to rename a dataset")
	}

	if scope.SourceName() != "local" {
		return nil, fmt.Errorf("can only rename using local source")
	}

	ref, err := dsref.ParseHumanFriendly(p.Current)
	// Allow bad upper-case characters in the left-hand side name, because it's needed to let users
	// fix badly named datasets.
	if err != nil && err != dsref.ErrBadCaseName {
		return nil, fmt.Errorf("original name: %w", err)
	}
	if _, err := scope.ResolveReference(scope.Context(), &ref); err != nil {
		return nil, err
	}

	next, err := dsref.ParseHumanFriendly(p.Next)
	if errors.Is(err, dsref.ErrNotHumanFriendly) {
		return nil, fmt.Errorf("destination name: %w", err)
	} else if err != nil {
		return nil, fmt.Errorf("destination name: %w", dsref.ErrDescribeValidName)
	}
	if ref.Username != next.Username && next.Username != "me" {
		return nil, fmt.Errorf("cannot change username or profileID of a dataset")
	}

	// Update the reference stored in the repo
	vi, err := base.RenameDatasetRef(scope.Context(), scope.Repo(), ref, next.Name)
	if err != nil {
		return nil, err
	}

	// If the dataset is linked to a working directory, update the ref
	if vi.FSIPath != "" {
		if _, err = scope.FSISubsystem().ModifyLinkReference(vi.FSIPath, vi.SimpleRef()); err != nil {
			return nil, err
		}
	}
	return vi, nil
}

// Remove a dataset entirely or remove a certain number of revisions
func (datasetImpl) Remove(scope scope, p *RemoveParams) (*RemoveResponse, error) {
	res := &RemoveResponse{}
	log.Debugf("Remove dataset ref %q, revisions %v", p.Ref, p.Revision)

	if p.Revision == nil {
		return nil, fmt.Errorf("invalid revision: nil")
	}

	if p.Revision.Gen == 0 {
		return nil, fmt.Errorf("invalid number of revisions to delete: 0")
	}
	if p.Revision.Field != "ds" {
		return nil, fmt.Errorf("can only remove whole dataset versions, not individual components")
	}
	if scope.SourceName() != "local" {
		return nil, fmt.Errorf("remove requires the 'local' source")
	}

	scope.EnableWorkingDir(true)
	ref, _, err := scope.ParseAndResolveRef(scope.Context(), p.Ref)
	if err != nil {
		log.Debugw("Remove, repo.ParseAndResolveRef failed", "err", err)
		// TODO(b5): this "logbook.ErrNotFound" is needed to get cmd.TestRemoveEvenIfLogbookGone
		// to pass. Relying on dataset resolution returning an error defined in logbook is incorrect
		// This should really be checking for some sort of "can't fully resolve" error
		// defined in dsref instead
		if p.Force || errors.Is(err, logbook.ErrNotFound) {
			didRemove, _ := base.RemoveEntireDataset(scope.Context(), scope.Repo(), ref, []dsref.VersionInfo{})
			if didRemove != "" {
				log.Debugw("Remove cleaned up data found", "didRemove", didRemove)
				res.Message = didRemove
				return res, nil
			}
		}
		return nil, err
	}

	res.Ref = ref.String()
	var fsiPath string
	if fsi.IsFSIPath(ref.Path) {
		fsiPath = fsi.FilesystemPathToLocal(ref.Path)
	}

	if fsiPath != "" {
		// Dataset is linked in a working directory.
		if !(p.KeepFiles || p.Force) {
			// Make sure that status is clean, otherwise, refuse to remove any revisions.
			// Setting either --keep-files or --force will skip this check.
			wdErr := scope.FSISubsystem().IsWorkingDirectoryClean(scope.Context(), fsiPath)
			if wdErr != nil {
				if wdErr == fsi.ErrWorkingDirectoryDirty {
					log.Debugf("Remove, IsWorkingDirectoryDirty")
					return nil, ErrCantRemoveDirectoryDirty
				}
				if errors.Is(wdErr, fsi.ErrNoLink) || strings.Contains(wdErr.Error(), "not a linked directory") {
					// If the working directory has been removed (or renamed), could not get the
					// status. However, don't let this stop the remove operation, since the files
					// are already gone, and therefore won't be removed.
					log.Debugf("Remove, couldn't get status for %s, maybe removed or renamed", fsiPath)
					wdErr = nil
				} else {
					log.Debugf("Remove, IsWorkingDirectoryClean error: %s", err)
					return nil, wdErr
				}
			}
		}
	} else if p.KeepFiles {
		// If dataset is not linked in a working directory, --keep-files can't be used.
		return nil, fmt.Errorf("dataset is not linked to filesystem, cannot use keep-files")
	}

	// Get the revisions that will be deleted.
	history, err := base.DatasetLog(scope.Context(), scope.Repo(), ref, p.Revision.Gen+1, 0, false)
	if err == nil && p.Revision.Gen >= len(history) {
		// If the number of revisions to delete is greater than or equal to the amount in history,
		// treat this operation as deleting everything.
		p.Revision.Gen = dsref.AllGenerations
	} else if err == repo.ErrNoHistory || errors.Is(err, dsref.ErrPathRequired) {
		// If the dataset has no history, treat this operation as deleting everything.
		p.Revision.Gen = dsref.AllGenerations
	} else if err != nil {
		log.Debugf("Remove, base.DatasetLog failed, error: %s", err)
		// Set history to a list of 0 elements. In the rest of this function, certain operations
		// check the history to figure out what to delete, they will always see a blank history,
		// which is a safer option for a destructive option such as remove.
		history = []dsref.VersionInfo{}
	}

	if p.Revision.Gen == dsref.AllGenerations {
		// removing all revisions of a dataset must unlink it
		if fsiPath != "" {
			if err := scope.FSISubsystem().Unlink(scope.Context(), fsiPath, ref); err == nil {
				res.Unlinked = true
			} else {
				log.Errorf("during Remove, dataset did not unlink: %s", err)
			}
		}

		didRemove, _ := base.RemoveEntireDataset(scope.Context(), scope.Repo(), ref, history)
		res.NumDeleted = dsref.AllGenerations
		res.Message = didRemove

		if fsiPath != "" && !p.KeepFiles {
			// Remove all files
			fsi.DeleteComponentFiles(fsiPath)
			var err error
			if p.Force {
				err = scope.FSISubsystem().RemoveAll(fsiPath)
			} else {
				err = scope.FSISubsystem().Remove(fsiPath)
			}
			if err != nil {
				if strings.Contains(err.Error(), "no such file or directory") {
					// If the working directory has already been removed (or renamed), it is
					// not an error that this remove operation fails, since we were trying to
					// remove them anyway.
					log.Debugf("Remove, couldn't remove %s, maybe already removed or renamed", fsiPath)
					err = nil
				} else {
					log.Debugf("Remove, os.Remove failed, error: %s", err)
					return nil, err
				}
			}
		}
	} else if len(history) > 0 {
		if fsiPath != "" {
			// if we're operating on an fsi-linked directory, we need to re-resolve to
			// get the path on qfs. This could be avoided if we refactored ParseAndResolveRef
			// to return an extra fsiPath value
			qfsRef := ref.Copy()
			qfsRef.Path = ""
			if _, err := scope.ResolveReference(scope.Context(), &qfsRef); err != nil {
				return nil, err
			}
			ref = qfsRef
		}
		// Delete the specific number of revisions.
		info, err := base.RemoveNVersionsFromStore(scope.Context(), scope.Repo(), ref, p.Revision.Gen)
		if err != nil {
			log.Debugf("Remove, base.RemoveNVersionsFromStore failed, error: %s", err)
			return nil, err
		}
		res.NumDeleted = p.Revision.Gen

		if info.FSIPath != "" && !p.KeepFiles {
			// Load dataset version that is at head after newer versions are removed
			ds, err := dsfs.LoadDataset(scope.Context(), scope.Filesystem(), info.Path)
			if err != nil {
				log.Debugf("Remove, dsfs.LoadDataset failed, error: %s", err)
				return nil, err
			}
			ds.Name = info.Name
			ds.Peername = info.Username
			if err = base.OpenDataset(scope.Context(), scope.Filesystem(), ds); err != nil {
				log.Debugf("Remove, base.OpenDataset failed, error: %s", err)
				return nil, err
			}

			// TODO(dlong): Add a method to FSI called ProjectOntoDirectory, use it here
			// and also for Restore() in lib/fsi.go and also maybe WriteComponents in fsi/mapping.go

			// Delete the old files
			err = fsi.DeleteComponentFiles(info.FSIPath)
			if err != nil {
				log.Debug("Remove, fsi.DeleteComponentFiles failed, error: %s", err)
			}

			// Update the files in the working directory
			fsi.WriteComponents(ds, info.FSIPath, scope.Filesystem())
		}
	}
	log.Debugf("Remove finished")

	return res, nil
}

// Pull downloads and stores an existing dataset to a peer's repository via
// a network connection
func (datasetImpl) Pull(scope scope, p *PullParams) (*dataset.Dataset, error) {
	res := &dataset.Dataset{}

	if scope.SourceName() != "network" {
		return nil, fmt.Errorf("pull requires the 'network' source")
	}

	ref, location, err := scope.ParseAndResolveRef(scope.Context(), p.Ref)
	if err != nil {
		log.Debugf("resolving reference: %s", err)
		return nil, err
	}
	log.Infof("pulling dataset from location: %s", location)

	ds, err := scope.RemoteClient().PullDataset(scope.Context(), &ref, location)
	if err != nil {
		log.Debugf("pulling dataset: %s", err)
		return nil, err
	}

	*res = *ds

	if p.LinkDir != "" {
		checkoutp := &LinkParams{
			Ref: ref.Human(),
			Dir: p.LinkDir,
		}
		// TODO (ramfox): wasn't sure exactly how to handle this. We don't need `Checkout` to
		// re-load/re-resolve the reference, but there is a bunch of other checking/verifying
		// that `Filesys().Checkout` does that doesn't belong in this `Pull` function
		// Should we allow method creation off of the `scope`? So in this case, `scope.Filesys()`
		// Or should we be obscuring all of that to create a `scope.Checkout()` method?
		if err = scope.inst.Filesys().Checkout(scope.Context(), checkoutp); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// Push posts a dataset version to a remote
func (datasetImpl) Push(scope scope, p *PushParams) (*dsref.Ref, error) {
	if scope.SourceName() != "local" {
		return nil, fmt.Errorf("push requires the 'local' source")
	}

	ref, _, err := scope.ParseAndResolveRef(scope.Context(), p.Ref)
	if err != nil {
		return nil, err
	}

	addr, err := remote.Address(scope.Config(), p.Remote)
	if err != nil {
		return nil, err
	}

	if err = scope.RemoteClient().PushDataset(scope.Context(), ref, addr); err != nil {
		return nil, err
	}

	if err = base.SetPublishStatus(scope.Context(), scope.Repo(), ref, true); err != nil {
		return nil, err
	}

	return &ref, nil
}

// Validate gives a dataset of errors and issues for a given dataset
func (datasetImpl) Validate(scope scope, p *ValidateParams) (*ValidateResponse, error) {
	res := &ValidateResponse{}

	if scope.SourceName() != "local" {
		return nil, fmt.Errorf("can only validate using local storage")
	}

	// Schema can come from either schema.json or structure.json, or the dataset itself.
	// schemaFlagType determines which of these three contains the schema.
	schemaFlagType := ""
	schemaFilename := ""
	if p.SchemaFilename != "" && p.StructureFilename != "" {
		return nil, qrierr.New(ErrBadArgs, "cannot provide both --schema and --structure flags")
	} else if p.SchemaFilename != "" {
		schemaFlagType = "schema"
		schemaFilename = p.SchemaFilename
	} else if p.StructureFilename != "" {
		schemaFlagType = "structure"
		schemaFilename = p.StructureFilename
	}

	if p.Ref == "" && (p.BodyFilename == "" || schemaFlagType == "") {
		return nil, qrierr.New(ErrBadArgs, "please provide a dataset name, or a supply the --body and --schema or --structure flags")
	}

	fsiPath := ""
	var err error
	ref := dsref.Ref{}

	// if there is both a bodyfilename and a schema/structure
	// we don't need to resolve any references
	if p.BodyFilename == "" || schemaFlagType == "" {
		scope.EnableWorkingDir(true)
		ref, _, err = scope.ParseAndResolveRef(scope.Context(), p.Ref)
		if err != nil {
			return nil, err
		}

		if fsi.IsFSIPath(ref.Path) {
			fsiPath = fsi.FilesystemPathToLocal(ref.Path)
		}
	}

	var ds *dataset.Dataset

	if p.Ref != "" {
		if fsiPath != "" {
			if ds, err = fsi.ReadDir(fsiPath); err != nil {
				return nil, fmt.Errorf("loading linked dataset: %w", err)
			}
		} else {
			if ds, err = dsfs.LoadDataset(scope.Context(), scope.Filesystem(), ref.Path); err != nil {
				return nil, fmt.Errorf("loading dataset: %w", err)
			}
		}
		if err = base.OpenDataset(scope.Context(), scope.Filesystem(), ds); err != nil {
			return nil, err
		}
	}

	var body qfs.File
	if p.BodyFilename == "" {
		body = ds.BodyFile()
	} else {
		// Body is set to the provided filename if given
		fs, err := localfs.NewFS(nil)
		if err != nil {
			return nil, fmt.Errorf("error creating new local filesystem: %w", err)
		}
		body, err = fs.Get(scope.Context(), p.BodyFilename)
		if err != nil {
			return nil, fmt.Errorf("error opening body file %q: %w", p.BodyFilename, err)
		}
	}

	var st *dataset.Structure
	// Schema is set to the provided filename if given, otherwise the dataset's schema
	if schemaFlagType == "" {
		st = ds.Structure
		if err := detect.Structure(ds); err != nil {
			log.Debug("lib.Validate: InferStructure error: %w", err)
			return nil, err
		}
	} else {
		data, err := ioutil.ReadFile(schemaFilename)
		if err != nil {
			return nil, fmt.Errorf("error opening %s file: %s", schemaFlagType, schemaFilename)
		}
		var fileContent map[string]interface{}
		err = json.Unmarshal(data, &fileContent)
		if err != nil {
			return nil, err
		}
		if schemaFlagType == "schema" {
			// If dataset ref was provided, get format from the structure. Otherwise, assume the
			// format by looking at the body file's extension.
			var bodyFormat string
			if ds != nil && ds.Structure != nil {
				bodyFormat = ds.Structure.Format
			} else {
				bodyFormat = filepath.Ext(p.BodyFilename)
				if strings.HasSuffix(bodyFormat, ".") {
					bodyFormat = bodyFormat[1:]
				}
			}
			st = &dataset.Structure{
				Format: bodyFormat,
				Schema: fileContent,
			}
		} else {
			// schemaFlagType == "structure". Deserialize the provided file into a structure.
			st = &dataset.Structure{}
			err = fill.Struct(fileContent, st)
			if err != nil {
				return nil, err
			}
			// TODO(dlong): What happens if body file extension does not match st.Format?
		}
	}

	valerrs, err := base.Validate(scope.Context(), scope.Repo(), body, st)
	if err != nil {
		return nil, err
	}

	*res = ValidateResponse{
		Structure: st,
		Errors:    valerrs,
	}
	return res, nil
}

// Manifest generates a manifest for a dataset path
func (datasetImpl) Manifest(scope scope, p *ManifestParams) (*dag.Manifest, error) {
	if scope.SourceName() != "local" {
		return nil, fmt.Errorf("can only create manifest using local storage")
	}

	res := &dag.Manifest{}
	ref, _, err := scope.ParseAndResolveRef(scope.Context(), p.Ref)
	if err != nil {
		return nil, err
	}

	res, err = scope.Node().NewManifest(scope.Context(), ref.Path)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ManifestMissing generates a manifest of blocks that are not present on this repo for a given manifes
func (datasetImpl) ManifestMissing(scope scope, p *ManifestMissingParams) (*dag.Manifest, error) {
	res := &dag.Manifest{}
	res, err := scope.Node().MissingManifest(scope.Context(), p.Manifest)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// DAGInfo generates a dag.Info for a dataset path. If a label is given, DAGInfo will generate a sub-dag.Info at that label.
func (datasetImpl) DAGInfo(scope scope, p *DAGInfoParams) (*dag.Info, error) {
	if scope.SourceName() != "local" {
		return nil, fmt.Errorf("can only create DAGInfo from local storage")
	}

	res := &dag.Info{}

	ref, _, err := scope.ParseAndResolveRef(scope.Context(), p.Ref)
	if err != nil {
		return nil, err
	}

	res, err = scope.Node().NewDAGInfo(scope.Context(), ref.Path, p.Label)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ComponentStatus gets changes that happened at a particular version in the history of the given
// dataset reference.
func (datasetImpl) ComponentStatus(scope scope, p *LinkParams) ([]StatusItem, error) {
	ctx := scope.Context()

	ref, _, err := scope.ParseAndResolveRef(ctx, p.Ref)
	if err != nil {
		return nil, err
	}

	return scope.FSISubsystem().StatusAtVersion(ctx, ref)
}

// Render renders a viz or readme component as html
func (datasetImpl) Render(scope scope, p *RenderParams) (res []byte, err error) {
	ds := p.Dataset
	if ds == nil {
		ds, err = scope.Loader().LoadDataset(scope.Context(), p.Ref)
		if err != nil {
			return nil, err
		}
	}

	switch p.Selector {
	case "viz":
		res, err = base.Render(scope.Context(), scope.Repo(), ds, p.Template)
		if err != nil {
			return nil, err
		}
	case "readme":
		if ds.Readme == nil {
			return nil, fmt.Errorf("no readme to render")
		}

		if err := ds.Readme.OpenScriptFile(scope.Context(), scope.Filesystem()); err != nil {
			return nil, err
		}
		if ds.Readme.ScriptFile() == nil {
			return nil, fmt.Errorf("no readme to render")
		}

		res, err = base.RenderReadme(scope.Context(), ds.Readme.ScriptFile())
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("selector must be one of 'viz' or 'readme'")
	}
	return res, nil
}
