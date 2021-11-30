package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/base/fill"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/lib"
	"gopkg.in/yaml.v2"
)

const (
	// maxBodyFileSize is 100MB
	maxBodyFileSize = 100 << 20
)

// GetBodyCSVHandler is a handler for returning the body as a csv file
// Examples:
// curl http://localhost:2503/ds/get/b5/world_bank_population/body.csv
func GetBodyCSVHandler(inst *lib.Instance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			util.NotFoundHandler(w, r)
			return
		}

		p := &lib.GetParams{}
		if err := UnmarshalParams(r, p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		p.Selector = "body"
		if err := validateCSVRequest(r, p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		outBytes, err := inst.Dataset().GetCSV(r.Context(), p)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}
		publishDownloadEvent(r.Context(), inst, p.Ref)
		writeFileResponse(w, outBytes, "body.csv", "csv")
	}
}

// GetHandler is a dataset single endpoint
func GetHandler(inst *lib.Instance, routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			util.NotFoundHandler(w, r)
			return
		}
		p := &lib.GetParams{}
		if err := UnmarshalParams(r, p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		format := r.FormValue("format")

		switch {
		case format == "csv", arrayContains(r.Header["Accept"], "text/csv"):
			// Examples:
			// curl http://localhost:2503/ds/get/b5/world_bank_population/body?format=csv
			// curl -H "Accept: text/csv" http://localhost:2503/ds/get/b5/world_bank_population/body
			if err := validateCSVRequest(r, p); err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			outBytes, err := inst.Dataset().GetCSV(r.Context(), p)
			if err != nil {
				util.RespondWithError(w, err)
				return
			}

			publishDownloadEvent(r.Context(), inst, p.Ref)
			writeFileResponse(w, outBytes, "body.csv", "csv")
			return

		case format == "zip", arrayContains(r.Header["Accept"], "application/zip"):
			// Examples:
			// curl -H "Accept: application/zip" http://localhost:2503/ds/get/world_bank_population
			// curl http://localhost:2503/ds/get/world_bank_population?format=zip
			if err := validateZipRequest(r, p); err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
			zipResults, err := inst.Dataset().GetZip(r.Context(), p)
			if err != nil {
				util.RespondWithError(w, err)
				return
			}
			publishDownloadEvent(r.Context(), inst, p.Ref)
			writeFileResponse(w, zipResults.Bytes, zipResults.GeneratedName, "zip")
			return

		default:
			res, err := inst.Dataset().Get(r.Context(), p)
			if err != nil {
				util.RespondWithError(w, err)
				return
			}

			if lib.IsSelectorScriptFile(p.Selector) {
				util.WriteResponse(w, res.Bytes)
				return
			}

			util.WriteResponse(w, res.Value)
		}
	}
}

func validateCSVRequest(r *http.Request, p *lib.GetParams) error {
	format := r.FormValue("format")
	if p.Selector != "body" {
		return fmt.Errorf("can only get csv of the body component, selector must be 'body'")
	}
	if !(format == "csv" || format == "") {
		return fmt.Errorf("format %q conflicts with requested body csv file", format)
	}
	return nil
}

func validateZipRequest(r *http.Request, p *lib.GetParams) error {
	format := r.FormValue("format")
	if p.Selector != "" {
		return fmt.Errorf("can only get zip file of the entire dataset, got selector %q", p.Selector)
	}
	if !(format == "zip" || format == "") {
		return fmt.Errorf("format %q conflicts with header %q", format, "Accept: application/zip")
	}
	return nil
}

// UnpackHandler unpacks a zip file and sends it back as json
func UnpackHandler(routePrefix string) http.HandlerFunc {
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

// SaveByUploadHandler saves a dataset by reading the body from a file
func SaveByUploadHandler(inst *lib.Instance, routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			util.NotFoundHandler(w, r)
			return
		}

		if err := r.ParseMultipartForm(maxBodyFileSize); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		p := &lib.SaveParams{}
		if err := parseSaveParamsFromRequest(r, p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		if p.Dataset == nil {
			p.Dataset = &dataset.Dataset{}
		}
		if err := formFileDataset(r, p.Dataset); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		if p.Dataset.BodyFile() != nil {
			// the `Save` method uses the `p.BodyPath` field to generate
			// a default dataset name name if one is not given in the ref
			p.BodyPath = p.Dataset.BodyFile().FileName()
		}

		ds, err := inst.Dataset().Save(r.Context(), p)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}
		util.WriteResponse(w, ds)
	}
}

// parseSaveParams parses the form from the request
// it ignores `bodyFile`, `filePaths`, and `secrets`
// `dataset`, if it exists, is expected to be a json string in
// the form of a `dataset.Dataset`
func parseSaveParamsFromRequest(r *http.Request, p *lib.SaveParams) error {
	p.Ref = r.FormValue("ref")
	p.Title = r.FormValue("title")
	p.Message = r.FormValue("message")
	p.Apply = util.ReqParamBool(r, "apply", false)
	p.Replace = util.ReqParamBool(r, "replace", false)
	p.Private = util.ReqParamBool(r, "private", false)
	p.ConvertFormatToPrev = util.ReqParamBool(r, "convertFormatToPrev", false)
	p.Drop = r.FormValue("drop")
	p.Force = util.ReqParamBool(r, "force", false)
	p.ShouldRender = util.ReqParamBool(r, "shouldRender", false)
	p.NewName = util.ReqParamBool(r, "newName", false)
	dsBytes := []byte(r.FormValue("dataset"))
	if len(dsBytes) != 0 {
		p.Dataset = &dataset.Dataset{}
		err := p.Dataset.UnmarshalJSON(dsBytes)
		if err != nil {
			return err
		}
	}
	log.Debugw("parseSaveParamsFromRequest", "params", p)
	return nil
}

// formFileDataset extracts a dataset document from a http Request
func formFileDataset(r *http.Request, ds *dataset.Dataset) (err error) {
	datafile, dataHeader, err := r.FormFile("file")
	if err == http.ErrMissingFile {
		// users should be allowed to only upload certain parts of the dataset
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
				err = fmt.Errorf("error decoding json file: %w", err)
				return
			}
		}
	}

	if username := r.FormValue("username"); username != "" {
		ds.Peername = username
	}
	if name := r.FormValue("name"); name != "" {
		ds.Name = name
	}
	if bp := r.FormValue("body_path"); bp != "" {
		ds.BodyPath = bp
	}

	tfFile, tfHeader, err := r.FormFile("transform")
	if err == http.ErrMissingFile {
		// users should be allowed to only upload certain parts of the dataset
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening transform file: %w", err)
		return
	}
	if tfFile != nil {
		if ds.Transform == nil {
			ds.Transform = &dataset.Transform{}
		}
		ds.Transform.SetScriptFile(qfs.NewMemfileReader(tfHeader.Filename, tfFile))
	}

	vizFile, vizHeader, err := r.FormFile("viz")
	if err == http.ErrMissingFile {
		// users should be allowed to only upload certain parts of the dataset
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening viz file: %w", err)
		return
	}
	if vizFile != nil {
		if ds.Viz == nil {
			ds.Viz = &dataset.Viz{}
		}
		ds.Viz.SetScriptFile(qfs.NewMemfileReader(vizHeader.Filename, vizFile))
	}

	readmeFile, readmeHeader, err := r.FormFile("readme")
	if err == http.ErrMissingFile {
		// users should be allowed to only upload certain parts of the dataset
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening readme file: %w", err)
		return
	}
	if readmeFile != nil {
		if ds.Readme == nil {
			ds.Readme = &dataset.Readme{}
			ds.Readme.SetScriptFile(qfs.NewMemfileReader(readmeHeader.Filename, readmeFile))
		}
	}

	bodyfile, bodyHeader, err := r.FormFile("body")
	if err == http.ErrMissingFile {
		// users should be allowed to only upload certain parts of the dataset
		err = nil
	} else if err != nil {
		err = fmt.Errorf("error opening body file: %w", err)
		return
	}
	if bodyfile != nil {
		ds.SetBodyFile(qfs.NewMemfileReader(bodyHeader.Filename, bodyfile))
	}

	return
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
	case ".md":
		return "text/x-markdown"
	case ".html":
		return "text/html"
	default:
		return ""
	}
}

func writeFileResponse(w http.ResponseWriter, val []byte, filename, format string) {
	w.Header().Set("Content-Type", extensionToMimeType("."+format))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(val)
}

func arrayContains(subject []string, target string) bool {
	for _, v := range subject {
		if v == target {
			return true
		}
	}
	return false
}

func publishDownloadEvent(ctx context.Context, inst *lib.Instance, refStr string) {
	ref, _, err := inst.ParseAndResolveRef(ctx, refStr, "local")
	if err != nil {
		log.Debugw("api.GetBodyCSVHandler - unable to resolve ref %q", err)
		return
	}
	inst.Bus().Publish(ctx, event.ETDatasetDownload, ref.InitID)
}
