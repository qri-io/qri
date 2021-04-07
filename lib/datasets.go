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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/gorilla/mux"
	"github.com/qri-io/dag"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/base/fill"
	"github.com/qri-io/qri/dscache/build"
	"github.com/qri-io/qri/dsref"
	qrierr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/fsi/linkfile"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
	"github.com/qri-io/qri/transform"
	"github.com/qri-io/qri/transform/run"
)

// DatasetMethods encapsulates business logic for working with Datasets on Qri
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
		"changereport": {AEChanges, "POST"},
		"daginfo":      {AEDAGInfo, "GET"},
		"diff":         {AEDiff, "GET"},
		"get":          {AEGet, "GET"},
		"list":         {AEList, "GET"},
		// TODO(dustmop): Needs its own endpoint
		"listrawrefs":     {AEList, "GET"},
		"manifest":        {AEManifest, "GET"},
		"manifestmissing": {AEManifestMissing, "GET"},
		"pull":            {AEPull, "POST"},
		"remove":          {AERemove, "POST"},
		"rename":          {AERename, "POST"},
		"render":          {AERender, "POST"},
		"save":            {AESave, "POST"},
		// TODO(dustmop): Needs its own endpoint
		"stats":    {AEGet, "GET"},
		"validate": {AEValidate, "GET"},
	}
}

// ErrListWarning is a warning that can occur while listing
var ErrListWarning = base.ErrUnlistableReferences

// List gets the reflist for either the local repo or a peer
func (m DatasetMethods) List(ctx context.Context, p *ListParams) ([]dsref.VersionInfo, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "list"), p)
	if res, ok := got.([]dsref.VersionInfo); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// ListRawRefs gets the list of raw references as string
func (m DatasetMethods) ListRawRefs(ctx context.Context, p *ListParams) (string, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "listrawrefs"), p)
	if res, ok := got.(string); ok {
		return res, err
	}
	return "", dispatchReturnError(got, err)
}

// GetParams defines parameters for looking up the head or body of a dataset
type GetParams struct {
	Refstr string `json:"ref"`

	Selector string `json:"selector"`

	// read from a filesystem link instead of stored version
	Format       string               `json:"format"`
	FormatConfig dataset.FormatConfig `json:"format_config"`

	Limit  int  `json:"limit"`
	Offset int  `json:"offset"`
	All    bool `json:"all"`

	// outfile is a filename to save the dataset to
	Outfile string `json:"outfile" qri:"fspath"`
	// whether to generate a filename from the dataset name instead
	GenFilename bool   `json:"genfilename"`
	Remote      string `json:"remote"`
}

// SetNonZeroDefaults assigns default values
func (p *GetParams) SetNonZeroDefaults() {
	if p.Format == "" {
		p.Format = "json"
	}
}

var validSelector = regexp.MustCompile(`^[\w-\.]*[\w]$`)

func parseSelector(selector string) (string, string, error) {
	if selector == "" {
		return "", "", nil
	}

	format := ""

	if strings.HasSuffix(selector, ".json") {
		format = "json"
	}
	if strings.HasSuffix(selector, ".csv") {
		format = "csv"
	}
	if strings.HasSuffix(selector, ".zip") {
		format = "zip"
	}

	if format != "" {
		selector = selector[:len(selector)-len(format)-1]
	}

	match := validSelector.FindString(selector)
	if match == "" || len(match) != len(selector) {
		return "", "", fmt.Errorf("could not parse request: invalid selector")
	}
	return selector, format, nil
}

func arrayContains(subject []string, target string) bool {
	for _, v := range subject {
		if v == target {
			return true
		}
	}
	return false
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *GetParams) UnmarshalFromRequest(r *http.Request) error {
	mvars := mux.Vars(r)

	if p == nil {
		p = &GetParams{}
	}

	params := *p
	if params.Refstr == "" {
		params.Refstr = r.FormValue("refstr")
	}

	ref, err := dsref.Parse(params.Refstr)
	if err != nil {
		return err
	}
	if ref.Username == "me" {
		return fmt.Errorf("username \"me\" not allowed")
	}

	if sel, ok := mvars["selector"]; ok && params.Selector == "" {
		selector, format, err := parseSelector(sel)
		if err != nil {
			return err
		}
		params.Selector = selector
		params.Format = format
	}

	if params.Format == "" {
		params.Format = r.FormValue("format")
	}

	// This HTTP header sets the format to csv, and removes the json wrapper
	if arrayContains(r.Header["Accept"], "text/csv") {
		if params.Format != "" && params.Format != "csv" {
			return fmt.Errorf("format %q conflicts with header \"Accept: text/csv\"", params.Format)
		}
		params.Format = "csv"
		params.Selector = "body"
	}

	if params.Format != "" && params.Format != "json" && params.Format != "csv" && params.Format != "zip" {
		return fmt.Errorf("invalid extension format")
	}

	if params.Remote == "" {
		params.Remote = r.FormValue("remote")
	}

	// TODO(arqu): we default to true but should implement a guard and/or respect the page params
	params.All = true
	// listParams := ListParamsFromRequest(r)
	// offset := listParams.Offset
	// limit := listParams.Limit
	// if offset == 0 && limit == -1 {
	// 	params.All = true
	// }

	*p = params
	return nil
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

// DataResponse is the struct used to respond to api requests made to the /body endpoint
// It is necessary because we need to include the 'path' field in the response
type DataResponse struct {
	Path string          `json:"path"`
	Data json.RawMessage `json:"data"`
}

// Get retrieves datasets and components for a given reference. p.Refstr is parsed to create
// a reference, which is used to load the dataset. It will be loaded from the local repo
// or from the filesystem if it has a linked working directory.
// Using p.Selector will control what components are returned in res.Bytes. The default,
// a blank selector, will also fill the entire dataset at res.Data. If the selector is "body"
// then res.Bytes is loaded with the body. If the selector is "stats", then res.Bytes is loaded
// with the generated stats.
func (m DatasetMethods) Get(ctx context.Context, p *GetParams) (*GetResult, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "get"), p)
	if res, ok := got.(*GetResult); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

func maybeWriteOutfile(p *GetParams, res *GetResult) error {
	if p.Outfile != "" {
		err := ioutil.WriteFile(p.Outfile, res.Bytes, 0644)
		if err != nil {
			return err
		}
		res.Bytes = []byte{}
	}
	return nil
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
	BodyPath string `qri:"fspath"`
	// absolute path or URL to the list of dataset files or components to load
	FilePaths []string `qri:"fspath"`
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
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *SaveParams) UnmarshalFromRequest(r *http.Request) error {

	if p.Dataset != nil || r.Header.Get("Content-Type") == "application/json" {
		if p.Dataset == nil {
			p.Dataset = &dataset.Dataset{}
		}
		pRef := p.Ref

		if pRef == "" {
			pRef = r.FormValue("refstr")
		}

		args, err := dsref.Parse(pRef)
		if err != nil {
			if err == dsref.ErrEmptyRef && r.FormValue("new") == "true" {
				err = nil
			} else {
				return err
			}
		}
		if args.Username != "" {
			p.Dataset.Peername = args.Username
			p.Dataset.Name = args.Name
		}
	} else {
		if p.Dataset == nil {
			p.Dataset = &dataset.Dataset{}
		}
		if err := formFileDataset(r, p.Dataset); err != nil {
			return err
		}
	}

	// TODO (b5) - this should probably be handled by lib
	// DatasetMethods.Save should fold the provided dataset values *then* attempt
	// to extract a valid dataset reference from the resulting dataset,
	// and use that as a save target.
	ref := reporef.DatasetRef{
		Name:     p.Dataset.Name,
		Peername: p.Dataset.Peername,
	}

	if p.Ref == "" {
		p.Ref = ref.AliasString()
	}
	if v := r.FormValue("apply"); v != "" {
		p.Apply = v == "true"
	}
	if v := r.FormValue("private"); v != "" {
		p.Private = v == "true"
	}
	if v := r.FormValue("force"); v != "" {
		p.Force = v == "true"
	}
	if v := r.FormValue("no_render"); v != "" {
		p.ShouldRender = !(v == "true")
	}
	if v := r.FormValue("new"); v != "" {
		p.NewName = v == "true"
	}
	if v := r.FormValue("bodypath"); v != "" {
		p.BodyPath = v
	}
	if v := r.FormValue("drop"); v != "" {
		p.Drop = v
	}

	if r.FormValue("secrets") != "" {
		p.Secrets = map[string]string{}
		if err := json.Unmarshal([]byte(r.FormValue("secrets")), &p.Secrets); err != nil {
			return fmt.Errorf("parsing secrets: %s", err)
		}
	} else if p.Dataset.Transform != nil && p.Dataset.Transform.Secrets != nil {
		// TODO remove this, require API consumers to send secrets separately
		p.Secrets = p.Dataset.Transform.Secrets
	}

	return nil
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
	Current, Next string
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
	Ref       string
	Revision  *dsref.Rev
	KeepFiles bool
	Force     bool
	Remote    string
}

// RemoveResponse gives the results of a remove
type RemoveResponse struct {
	Ref        string
	NumDeleted int
	Message    string
	Unlinked   bool
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *RemoveParams) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &RemoveParams{}
	}

	if p.Ref == "" {
		p.Ref = r.FormValue("refstr")
	}
	if p.Remote == "" {
		p.Remote = r.FormValue("remote")
	}
	if p.KeepFiles == false {
		p.KeepFiles = r.FormValue("keep-files") == "true"
	}
	if p.Force == false {
		p.Force = r.FormValue("force") == "true"
	}

	if r.FormValue("all") == "true" {
		p.Revision = dsref.NewAllRevisions()
	}

	return nil
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
	Ref      string
	LinkDir  string `qri:"fspath"`
	Remote   string // remote to attempt to pull from
	LogsOnly bool   // only fetch logbook data
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *PullParams) UnmarshalFromRequest(r *http.Request) error {
	if p.Ref == "" {
		p.Ref = r.FormValue("refstr")
	}

	return nil
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

// ValidateParams defines parameters for dataset data validation
type ValidateParams struct {
	Ref               string
	BodyFilename      string
	SchemaFilename    string
	StructureFilename string
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *ValidateParams) UnmarshalFromRequest(r *http.Request) error {
	if p.Ref == "" {
		p.Ref = r.FormValue("refstr")
	}

	return nil
}

// ValidateResponse is the result of running validate against a dataset
type ValidateResponse struct {
	// Structure used to perform validation
	Structure *dataset.Structure
	// Validation Errors
	Errors []jsonschema.KeyError
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
	Refstr string
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
	Manifest *dag.Manifest
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
	RefStr, Label string
}

// DAGInfo generates a dag.Info for a dataset path. If a label is given, DAGInfo will generate a sub-dag.Info at that label.
func (m DatasetMethods) DAGInfo(ctx context.Context, p *DAGInfoParams) (*dag.Info, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "daginfo"), p)
	if res, ok := got.(*dag.Info); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// StatsParams defines the params for a Stats request
type StatsParams struct {
	// string representation of a dataset reference
	Refstr string
	// if we get a Dataset from the params, then we do not have to
	// attempt to open a dataset from the reference
	Dataset *dataset.Dataset
}

// Stats generates stats for a dataset
func (m DatasetMethods) Stats(ctx context.Context, p *StatsParams) (*dataset.Stats, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "stats"), p)
	if res, ok := got.(*dataset.Stats); ok {
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
			format, err := detect.ExtensionDataFormat(bodyHeader.Filename)
			if err != nil {
				return err
			}
			st, _, err := detect.FromReader(format, bytes.NewReader(ds.BodyBytes))
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
	Ref string
	// Optionally pass an entire dataset in for rendering, if providing a dataset,
	// the Ref field must be empty
	Dataset *dataset.Dataset
	// Optional template override
	Template []byte
	// If true,
	UseFSI bool
	// Output format. defaults to "html"
	Format string
	// remote resolver to use
	Remote string
	// Selector
	Selector string
}

// SetNonZeroDefaults assigns default values
func (p *RenderParams) SetNonZeroDefaults() {
	if p.Format == "" {
		p.Format = "html"
	}
}

// UnmarshalFromRequest implements a custom deserialization-from-HTTP request
func (p *RenderParams) UnmarshalFromRequest(r *http.Request) error {
	if p == nil {
		p = &RenderParams{}
	}

	params := *p
	if params.Ref == "" {
		params.Ref = r.FormValue("refstr")
	}

	_, err := dsref.Parse(params.Ref)
	if err != nil && params.Dataset == nil {
		return err
	}

	if params.Selector == "" {
		params.Selector = r.FormValue("selector")
	}

	if !params.UseFSI {
		params.UseFSI = r.FormValue("fsi") == "true"
	}

	if params.Remote == "" {
		params.Remote = r.FormValue("remote")
	}
	if params.Format == "" {
		params.Format = r.FormValue("format")
	}

	*p = params
	return nil
}

// Validate checks if render parameters are valid
func (p *RenderParams) Validate() error {
	if p.Ref != "" && p.Dataset != nil {
		return fmt.Errorf("cannot provide both a reference and a dataset to render")
	}
	if p.Ref == "" && p.Dataset == nil {
		return fmt.Errorf("must provide either a dataset or a dataset reference")
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

// List gets the reflist for either the local repo or a peer
func (datasetImpl) List(scope scope, p *ListParams) ([]dsref.VersionInfo, error) {
	// TODO(dustmop): When List is converted to use scope, get the ProfileID from
	// the scope if the user is authorized to only view their own datasets, as opposed
	// to the full collection that exists in this node's repository.
	restrictPid := ""

	// ensure valid limit value
	if p.Limit <= 0 {
		p.Limit = 25
	}
	// ensure valid offset value
	if p.Offset < 0 {
		p.Offset = 0
	}

	reqProfile := scope.Repo().Profiles().Owner()
	listProfile, err := getProfile(scope.Context(), scope.Repo().Profiles(), reqProfile.ID.String(), p.Peername)
	if err != nil {
		return nil, err
	}

	// If the list operation leads to a warning, store it in this var
	var listWarning error

	var infos []dsref.VersionInfo
	if p.UseDscache {
		c := scope.Dscache()
		if c.IsEmpty() {
			log.Infof("building dscache from repo's logbook, profile, and dsref")
			built, err := build.DscacheFromRepo(scope.Context(), scope.Repo())
			if err != nil {
				return nil, err
			}
			err = c.Assign(built)
			if err != nil {
				log.Error(err)
			}
		}
		refs, err := c.ListRefs()
		if err != nil {
			return nil, err
		}
		// Filter references so that only with a matching name are returned
		if p.Term != "" {
			matched := make([]reporef.DatasetRef, len(refs))
			count := 0
			for _, ref := range refs {
				if strings.Contains(ref.AliasString(), p.Term) {
					matched[count] = ref
					count++
				}
			}
			refs = matched[:count]
		}
		// Filter references by skipping to the correct offset
		if p.Offset > len(refs) {
			refs = []reporef.DatasetRef{}
		} else {
			refs = refs[p.Offset:]
		}
		// Filter references by limiting how many are returned
		if p.Limit < len(refs) {
			refs = refs[:p.Limit]
		}
		// Convert old style DatasetRef list to VersionInfo list.
		// TODO(dustmop): Remove this and convert lower-level functions to return []VersionInfo.
		infos = make([]dsref.VersionInfo, len(refs))
		for i, r := range refs {
			infos[i] = reporef.ConvertToVersionInfo(&r)
		}
	} else if listProfile.Peername == "" || reqProfile.Peername == listProfile.Peername {
		infos, err = base.ListDatasets(scope.Context(), scope.Repo(), p.Term, restrictPid, p.Offset, p.Limit, p.RPC, p.Public, p.ShowNumVersions)
		if errors.Is(err, ErrListWarning) {
			listWarning = err
			err = nil
		}
	} else {
		return nil, fmt.Errorf("listing datasets on a peer is not implemented")
	}
	if err != nil {
		return nil, err
	}

	if p.EnsureFSIExists {
		// For each reference with a linked fsi working directory, check that the folder exists
		// and has a .qri-ref file. If it's missing, remove the link from the centralized repo.
		// Doing this every list operation is a bit inefficient, so the behavior is opt-in.
		for _, info := range infos {
			if info.FSIPath != "" && !linkfile.ExistsInDir(info.FSIPath) {
				info.FSIPath = ""
				ref := reporef.RefFromVersionInfo(&info)
				if ref.Path == "" {
					if err = scope.Repo().DeleteRef(ref); err != nil {
						log.Debugf("cannot delete ref for %q, err: %s", ref, err)
					}
					continue
				}
				if err = scope.Repo().PutRef(ref); err != nil {
					log.Debugf("cannot put ref for %q, err: %s", ref, err)
				}
			}
		}
	}

	if listWarning != nil {
		return nil, listWarning
	}

	return infos, nil
}

func getProfile(ctx context.Context, pros profile.Store, idStr, peername string) (pro *profile.Profile, err error) {
	if idStr == "" {
		// TODO(b5): we're handling the "me" keyword here, should be handled as part of
		// request scope construction
		if peername == "me" {
			return pros.Owner(), nil
		}
		return profile.ResolveUsername(pros, peername)
	}

	id, err := profile.IDB58Decode(idStr)
	if err != nil {
		log.Debugw("decoding profile ID", "err", err)
		return nil, err
	}
	return pros.GetProfile(id)
}

// ListRawRefs gets the list of raw references as string
func (datasetImpl) ListRawRefs(scope scope, p *ListParams) (string, error) {
	text := ""
	if p.UseDscache {
		c := scope.Dscache()
		if c == nil || c.IsEmpty() {
			return "", fmt.Errorf("repo: dscache not found")
		}
		text = c.VerboseString(true)
		return text, nil
	}
	return base.RawDatasetRefs(scope.Context(), scope.Repo())
}

// Get retrieves datasets and components for a given reference.t
func (datasetImpl) Get(scope scope, p *GetParams) (*GetResult, error) {
	res := &GetResult{}

	var ds *dataset.Dataset
	ref, source, err := scope.ParseAndResolveRefWithWorkingDir(scope.Context(), p.Refstr, p.Remote)
	if err != nil {
		return nil, err
	}
	ds, err = scope.LoadDataset(scope.Context(), ref, source)
	if err != nil {
		return nil, err
	}

	res.Ref = &ref
	res.Dataset = ds

	if fsi.IsFSIPath(ref.Path) {
		res.FSIPath = fsi.FilesystemPathToLocal(ref.Path)
	}
	// TODO (b5) - Published field is longer set as part of Reference Resolution
	// getting publication status should be delegated to a new function

	if err = base.OpenDataset(scope.Context(), scope.Filesystem(), ds); err != nil {
		log.Debugf("Get dataset, base.OpenDataset failed, error: %s", err)
		return nil, err
	}

	if p.Format == "zip" {
		// Only if GenFilename is true, and no output filename is set, generate one from the
		// dataset name
		if p.Outfile == "" && p.GenFilename {
			p.Outfile = fmt.Sprintf("%s.zip", ds.Name)
		}
		var outBuf bytes.Buffer
		var zipFile io.Writer
		if p.Outfile == "" {
			// In this case, write to a buffer, which will be assigned to res.Bytes later on
			zipFile = &outBuf
		} else {
			zipFile, err = os.Create(p.Outfile)
			if err != nil {
				return nil, err
			}
		}
		currRef := dsref.Ref{Username: ds.Peername, Name: ds.Name}
		// TODO(dustmop): This function is inefficient and a poor use of logbook, but it's
		// necessary until dscache is in use.
		initID, err := scope.Logbook().RefToInitID(currRef)
		if err != nil {
			return nil, err
		}
		err = archive.WriteZip(scope.Context(), scope.Filesystem(), ds, "json", initID, currRef, zipFile)
		if err != nil {
			return nil, err
		}
		// Handle output. If outfile is empty, return the raw bytes. Otherwise provide a helpful
		// message for the user
		if p.Outfile == "" {
			res.Bytes = outBuf.Bytes()
		} else {
			res.Message = fmt.Sprintf("Wrote archive %s", p.Outfile)
		}
		return res, nil
	}

	if p.Selector == "body" {
		// `qri get body` loads the body
		if !p.All && (p.Limit < 0 || p.Offset < 0) {
			return nil, fmt.Errorf("invalid limit / offset settings")
		}
		df, err := dataset.ParseDataFormatString(p.Format)
		if err != nil {
			log.Debugf("Get dataset, ParseDataFormatString %q failed, error: %s", p.Format, err)
			return nil, err
		}

		if fsi.IsFSIPath(ref.Path) {
			// TODO(dustmop): Need to handle the special case where an FSI directory has a body
			// but no structure, which should infer a schema in order to read the body. Once that
			// works we can remove the fsi.GetBody call and just use base.ReadBody.
			res.Bytes, err = fsi.GetBody(fsi.FilesystemPathToLocal(ref.Path), df, p.FormatConfig, p.Offset, p.Limit, p.All)
			if err != nil {
				log.Debugf("Get dataset, fsi.GetBody %q failed, error: %s", res.FSIPath, err)
				return nil, err
			}
			err = maybeWriteOutfile(p, res)
			if err != nil {
				return nil, err
			}
			return res, nil
		}
		res.Bytes, err = base.ReadBody(ds, df, p.FormatConfig, p.Limit, p.Offset, p.All)
		if err != nil {
			log.Debugf("Get dataset, base.ReadBody %q failed, error: %s", ds, err)
			return nil, err
		}
		err = maybeWriteOutfile(p, res)
		if err != nil {
			return nil, err
		}
		return res, nil
	} else if scriptFile, ok := scriptFileSelection(ds, p.Selector); ok {
		// Fields that have qfs.File types should be read and returned
		res.Bytes, err = ioutil.ReadAll(scriptFile)
		if err != nil {
			return nil, err
		}
		err = maybeWriteOutfile(p, res)
		if err != nil {
			return nil, err
		}
		return res, nil
	} else if p.Selector == "stats" {
		sa, err := scope.Stats().Stats(scope.Context(), res.Dataset)
		if err != nil {
			return nil, err
		}
		res.Bytes, err = json.Marshal(sa.Stats)
		if err != nil {
			return nil, err
		}
		err = maybeWriteOutfile(p, res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	var value interface{}
	if p.Selector == "" {
		// `qri get` without a selector loads only the dataset head
		value = res.Dataset
	} else {
		// `qri get <selector>` loads only the applicable component / field
		value, err = base.ApplyPath(res.Dataset, p.Selector)
		if err != nil {
			return nil, err
		}
	}
	switch p.Format {
	case "json":
		// Pretty defaults to true for the dataset head, unless explicitly set in the config.
		pretty := true
		if p.FormatConfig != nil {
			pvalue, ok := p.FormatConfig.Map()["pretty"].(bool)
			if ok {
				pretty = pvalue
			}
		}
		if pretty {
			res.Bytes, err = json.MarshalIndent(value, "", " ")
			if err != nil {
				return nil, err
			}
		} else {
			res.Bytes, err = json.Marshal(value)
			if err != nil {
				return nil, err
			}
		}
	case "yaml", "":
		res.Bytes, err = yaml.Marshal(value)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown format: \"%s\"", p.Format)
	}
	err = maybeWriteOutfile(p, res)
	if err != nil {
		return nil, err
	}
	return res, nil
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
	if p.UseDscache {
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
		// allocate an ID for the transform, subscribe to print output & build up
		// runState
		runID := run.NewID()
		runState = run.NewState(runID)
		// create a loader so transforms can call `load_dataset`
		// TODO(b5) - add a ResolverMode save parameter and call m.d.resolverForMode
		// on the passed in mode string instead of just using the default resolver
		// cmd can then define "remote" and "offline" flags, that set the ResolverMode
		// string and control how transform functions
		loader := scope.ParseResolveFunc()

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
		transformer := transform.NewTransformer(scope.AppContext(), loader, scope.Bus())
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

	ref, err := dsref.ParseHumanFriendly(p.Current)
	// Allow bad upper-case characters in the left-hand side name, because it's needed to let users
	// fix badly named datasets.
	if err != nil && err != dsref.ErrBadCaseName {
		return nil, fmt.Errorf("original name: %w", err)
	}
	if _, err := scope.ResolveReference(scope.Context(), &ref, "local"); err != nil {
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

	if p.Revision.Gen == 0 {
		return nil, fmt.Errorf("invalid number of revisions to delete: 0")
	}
	if p.Revision.Field != "ds" {
		return nil, fmt.Errorf("can only remove whole dataset versions, not individual components")
	}

	ref, _, err := scope.ParseAndResolveRefWithWorkingDir(scope.Context(), p.Ref, "local")
	if err != nil {
		log.Debugw("Remove, repo.ParseAndResolveRefWithWorkingDir failed", "err", err)
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
			// get the path on qfs. This could be avoided if we refactored ParseAndResolveRefWithWorkingDir
			// to return an extra fsiPath value
			qfsRef := ref.Copy()
			qfsRef.Path = ""
			if _, err := scope.ResolveReference(scope.Context(), &qfsRef, "local"); err != nil {
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
	// TODO(dustmop): source has moved to the scope, and passing it to ParseAndResolveRef
	// does nothing. Remove it from here and from the third parameter of that func
	source := p.Remote
	if source == "" {
		source = "network"
	}

	ref, location, err := scope.ParseAndResolveRef(scope.Context(), p.Ref, source)
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
			Refstr: ref.Human(),
			Dir:    p.LinkDir,
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

// Validate gives a dataset of errors and issues for a given dataset
func (datasetImpl) Validate(scope scope, p *ValidateParams) (*ValidateResponse, error) {
	res := &ValidateResponse{}

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
		// TODO (ramfox): we need consts in `dsref` for "local", "network", "p2p"
		ref, _, err = scope.ParseAndResolveRefWithWorkingDir(scope.Context(), p.Ref, "local")
		if err != nil {
			return nil, err
		}

		if fsi.IsFSIPath(ref.Path) {
			fsiPath = fsi.FilesystemPathToLocal(ref.Path)
		}
	}

	var ds *dataset.Dataset

	// TODO(dlong): This pattern has shown up many places, such as lib.Get.
	// Should probably combine into a utility function.

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
	res := &dag.Manifest{}
	ref, _, err := scope.ParseAndResolveRef(scope.Context(), p.Refstr, "local")
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
	res := &dag.Info{}

	ref, _, err := scope.ParseAndResolveRef(scope.Context(), p.RefStr, "local")
	if err != nil {
		return nil, err
	}

	res, err = scope.Node().NewDAGInfo(scope.Context(), ref.Path, p.Label)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Stats generates stats for a dataset
func (datasetImpl) Stats(scope scope, p *StatsParams) (*dataset.Stats, error) {
	if p.Refstr == "" && p.Dataset == nil {
		return nil, fmt.Errorf("either a reference or dataset is required")
	}

	ds := p.Dataset
	if ds == nil {
		// TODO (b5) - stats is currently local-only, supply a source parameter
		ref, source, err := scope.ParseAndResolveRefWithWorkingDir(scope.Context(), p.Refstr, "local")
		if err != nil {
			return nil, err
		}
		if ds, err = scope.LoadDataset(scope.Context(), ref, source); err != nil {
			return nil, err
		}
	}

	return scope.Stats().Stats(scope.Context(), ds)
}

// Render renders a viz or readme component as html
func (datasetImpl) Render(scope scope, p *RenderParams) (res []byte, err error) {
	ds := p.Dataset
	if ds == nil {
		ref, source, err := scope.ParseAndResolveRefWithWorkingDir(scope.Context(), p.Ref, p.Remote)
		if err != nil {
			return nil, err
		}
		ds, err = scope.LoadDataset(scope.Context(), ref, source)
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
