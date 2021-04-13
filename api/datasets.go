package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/lib"
)

// DatasetHandlers wraps a requests struct to interface with http.HandlerFunc
type DatasetHandlers struct {
	lib.DatasetMethods
	inst     *lib.Instance
	ReadOnly bool
}

// NewDatasetHandlers allocates a DatasetHandlers pointer
func NewDatasetHandlers(inst *lib.Instance, readOnly bool) *DatasetHandlers {
	h := DatasetHandlers{inst.Dataset(), inst, readOnly}
	return &h
}

// GetHandler is a dataset single endpoint
func (h *DatasetHandlers) GetHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == http.MethodGet || r.Method == http.MethodPost) {
			util.NotFoundHandler(w, r)
			return
		}
		params := &lib.GetParams{}

		err := lib.UnmarshalParams(r, params)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		got, _, err := h.inst.Dispatch(r.Context(), "dataset.get", params)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}
		res, ok := got.(*lib.GetResult)
		if !ok {
			util.RespondWithDispatchTypeError(w, got)
			return
		}
		h.replyWithGetResponse(w, r, params, res)
	}
}

// UnpackHandler unpacks a zip file and sends it back as json
func (h *DatasetHandlers) UnpackHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			util.NotFoundHandler(w, r)
			return
		}
		postData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		contents, err := archive.UnzipGetContents(postData)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		data, err := json.Marshal(contents)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		util.WriteResponse(w, json.RawMessage(data))
	}
}

func extensionToMimeType(ext string) string {
	switch ext {
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	case ".yaml":
		return "application/x-yaml"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".zip":
		return "application/zip"
	case ".txt":
		return "text/plain"
	default:
		return ""
	}
}

// inlineScriptsToBytes consumes all open script files for dataset components
// other than the body, inlining file data to scriptBytes fields
func inlineScriptsToBytes(ds *dataset.Dataset) error {
	var err error
	if ds.Readme != nil && ds.Readme.ScriptFile() != nil {
		if ds.Readme.ScriptBytes, err = ioutil.ReadAll(ds.Readme.ScriptFile()); err != nil {
			return err
		}
	}

	if ds.Transform != nil && ds.Transform.ScriptFile() != nil {
		if ds.Transform.ScriptBytes, err = ioutil.ReadAll(ds.Transform.ScriptFile()); err != nil {
			return err
		}
	}

	if ds.Viz != nil && ds.Viz.ScriptFile() != nil {
		if ds.Viz.ScriptBytes, err = ioutil.ReadAll(ds.Viz.ScriptFile()); err != nil {
			return err
		}
	}

	return nil
}

// replyWithGetResponse writes an http response back to the client, based upon what sort of
// response they requested. Handles raw file downloads (without response wrappers), zip downloads,
// body pagination, as well as normal head responses. Input logic has already been handled
// before this function, so errors should not commonly happen.
func (h *DatasetHandlers) replyWithGetResponse(w http.ResponseWriter, r *http.Request, params *lib.GetParams, result *lib.GetResult) {
	resultFormat := params.Format
	if resultFormat == "" {
		resultFormat = result.Dataset.Structure.Format
	}

	if resultFormat == "json" {
		// Convert components with scriptPaths (transform, readme, viz) in scriptBytes
		if err := inlineScriptsToBytes(result.Dataset); err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}

		if params.Selector != "" {
			page := util.PageFromRequest(r)
			dataResponse := lib.DataResponse{
				Path: result.Dataset.BodyPath,
				Data: json.RawMessage(result.Bytes),
			}
			stripServerSideQueryParams(r)
			if err := util.WritePageResponse(w, dataResponse, r, page); err != nil {
				log.Infof("error writing response: %s", err.Error())
			}
			return
		}
		util.WriteResponse(w, result)
	} else {
		filename, err := archive.GenerateFilename(result.Dataset, resultFormat)
		if err != nil {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", extensionToMimeType("."+resultFormat))
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Write(result.Bytes)
	}
}

func loadFileIfPath(path string) (file *os.File, err error) {
	if path == "" {
		return nil, nil
	}

	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("filepath must be absolute")
	}

	return os.Open(path)
}
