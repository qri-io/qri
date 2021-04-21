package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/lib"
)

// GetHandler is a dataset single endpoint
func GetHandler(inst *lib.Instance, routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			util.NotFoundHandler(w, r)
			return
		}

		var err error
		returnCase, params, err := getCaseAndParamsFromRequest(r)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		switch returnCase {

		case "zip":
			outBytes, err := inst.Dataset().GetZip(r.Context(), params)
			if err != nil {
				util.RespondWithError(w, err)
				return
			}
			writeFileResponse(w, outBytes, "dataset.zip", "zip")
			return

		case "csv":
			outBytes, err := inst.Dataset().GetCSV(r.Context(), params)
			if err != nil {
				util.RespondWithError(w, err)
				return
			}
			writeFileResponse(w, outBytes, "body.csv", "csv")
			return

		case "text/json":
			res, err := inst.Dataset().Get(r.Context(), params)
			if err != nil {
				util.RespondWithError(w, err)
			}
			outbytes, err := json.Marshal(res.Value)
			if err != nil {
				util.RespondWithError(w, err)
			}
			componentName := params.Selector
			if componentName == "" {
				componentName = "dataset"
			}
			writeFileResponse(w, outbytes, fmt.Sprintf("%s.json", componentName), "json")
			return

		case "pretty":
			res, err := inst.Dataset().Get(r.Context(), params)
			if err != nil {
				util.RespondWithError(w, err)
			}
			outbytes, err := json.MarshalIndent(res.Value, "", "  ")
			if err != nil {
				util.RespondWithError(w, err)
			}
			componentName := params.Selector
			if componentName == "" {
				componentName = "dataset"
			}
			writeFileResponse(w, outbytes, fmt.Sprintf("%s.json", componentName), "json")
			return
		}

		res, err := inst.Dataset().Get(r.Context(), params)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}

		if lib.IsSelectorScriptFile(params.Selector) {
			util.WriteResponse(w, res.Bytes)
			return
		}

		util.WriteResponse(w, res.Value)
	}
}

// possible return cases:
//  - "zip"
//  - "csv"
//  - "pretty"
//  - "text/json"
//  - ""
func getCaseAndParamsFromRequest(r *http.Request) (string, *lib.GetParams, error) {
	p := &lib.GetParams{}
	log.Debugf("ref:%s", r.FormValue("ref"))
	p.Ref = r.FormValue("ref")

	ref, err := dsref.Parse(p.Ref)
	if err != nil {
		return "", nil, err
	}

	if ref.Username == "me" {
		return "", nil, fmt.Errorf("username \"me\" not allowed")
	}

	format := r.FormValue("format")
	p.Selector = r.FormValue("selector")

	p.All = util.ReqParamBool(r, "all", false)

	// TODO(arqu): we default to true but should implement a guard and/or respect the page params
	listParams := lib.ListParamsFromRequest(r)
	offset := listParams.Offset
	limit := listParams.Limit
	if offset == 0 && limit == -1 {
		p.All = true
	}

	switch {
	case format == "csv":
		fallthrough
	case arrayContains(r.Header["Accept"], "text/csv"):
		if !(format == "csv" || format == "") {
			return "", nil, fmt.Errorf("format %q conflicts with header %q", format, "Accept: text/csv")
		}
		fallthrough
	case p.Selector == "body.csv":
		// curl -H "Accept: text/csv" http://localhost:2503/ds/get/b5/world_bank_population/body
		// curl http://localhost:2503/ds/get/b5/world_bank_population/body.csv
		if !(p.Selector == "body" || p.Selector == "body.csv") {
			return "", nil, fmt.Errorf("can only get csv of the body component, selector must be 'body'")
		}
		p.Selector = "body"
		if p.Limit == 0 && p.Offset == 0 {
			p.All = true
		}
		return "csv", p, nil

	case format == "pretty":
		fallthrough
	case arrayContains(r.Header["Accept"], "text/json"):
		if !(format == "json" || format == "pretty" || format == "") {
			return "", nil, fmt.Errorf("format %q conflicts with header %q", format, "Accept: text/json")
		}
		fallthrough
	case strings.HasSuffix(p.Selector, ".json"):
		// curl -H "Accept: text/json" http://localhost:2503/ds/get/b5/world_bank_population
		// curl -H "Accept: text/json" http://localhost:2503/ds/get/b5/world_bank_population/body
		// curl -H "Accept: text/json" http://localhost:2503/ds/get/b5/world_bank_population/meta
		// curl -H "Accept: text/json" http://localhost:2503/ds/get/b5/world_bank_population/meta?pretty=true
		// curl http://localhost:2503/ds/get/b5/world_bank_population/body.json
		// curl http://localhost:2503/ds/get/b5/world_bank_population/body.json?pretty=true
		if p.Selector == "dataset.json" {
			p.Selector = ""
		}
		if strings.HasSuffix(p.Selector, ".json") {
			p.Selector = strings.TrimSuffix(p.Selector, ".json")
		}
		if format == "pretty" {
			return "pretty", p, nil
		}
		return "text/json", p, nil

	case format == "zip":
		fallthrough
	case arrayContains(r.Header["Accept"], "application/zip"):
		if !(format == "zip" || format == "") {
			return "", nil, fmt.Errorf("format %q conflicts with header %q", format, "Accept: application/zip")
		}
		fallthrough
	case p.Selector == "dataset.zip":
		// curl -H "Accept: application/zip" http://localhost:2503/ds/get/world_bank_population
		// curl http://localhost:2503/ds/get/world_bank_population/dataset.zip
		if !(p.Selector == "dataset.zip" || p.Selector == "") {
			return "", nil, fmt.Errorf("can only get zip file of entire dataset")
		}
		p.Selector = ""
		return "zip", p, nil

	default:
		return "", p, nil
	}
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
