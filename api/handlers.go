package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/base/archive"
	"github.com/qri-io/qri/lib"
)

// GetHandler is a dataset single endpoint
func GetHandler(inst *lib.Instance, routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			util.NotFoundHandler(w, r)
			return
		}
		params := &lib.GetParams{}

		var err error
		err = UnmarshalParams(r, params)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		switch {
		// case params.Format == "zip":
		// 	outBytes, err := lib.GetZip(r.Context(), inst, params.Ref)
		// 	if err != nil {
		// 		util.RespondWithError(w, err)
		// 		return
		// 	}
		// 	writeFileResponse(w, outBytes, "dataset.zip", "zip")
		// 	return
		// case params.Format == "csv":
		// 	outBytes, err := lib.GetCSV(r.Context(), inst, params.Ref, params.Limit, params.Offset, params.All)
		// 	if err != nil {
		// 		util.RespondWithError(w, err)
		// 		return
		// 	}
		// 	writeFileResponse(w, outBytes, "body.csv", "csv")
		// 	return

		case lib.IsSelectorScriptFile(params.Selector):
			res, err := inst.Dataset().Get(r.Context(), params)
			if err != nil {
				util.RespondWithError(w, err)
				return
			}
			var ok bool
			outBytes, ok := res.Value.([]byte)
			if !ok {
				util.RespondWithError(w, fmt.Errorf("error getting script file"))
				return
			}
			switch {
			case strings.Contains(params.Selector, "readme"):
				writeFileResponse(w, outBytes, "readme.md", "md")
				return
			case strings.Contains(params.Selector, "viz"):
				writeFileResponse(w, outBytes, "viz.html", "html")
				return
			case strings.Contains(params.Selector, "transform"):
				writeFileResponse(w, outBytes, "transform.star", "star")
				return
			default:
				util.RespondWithError(w, fmt.Errorf("given component does not have a scriptfile"))
				return
			}
			return
		}
		res, err := inst.Dataset().Get(r.Context(), params)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}

		util.WriteResponse(w, res.Value)
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
