package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/lib"
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
