package api

import (
	"encoding/json"
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
)

// SQLHandlers connects HTTP requests to the FSI subsystem
type SQLHandlers struct {
	lib.SQLMethods
	ReadOnly bool
}

// NewSQLHandlers creates handlers that talk to qri's filesystem integration
func NewSQLHandlers(inst *lib.Instance, readOnly bool) SQLHandlers {
	return SQLHandlers{
		SQLMethods: *lib.NewSQLMethods(inst),
		ReadOnly:   readOnly,
	}
}

// QueryHandler runs an SQL query over HTTP
func (h *SQLHandlers) QueryHandler(routePrefix string) http.HandlerFunc {
	handleQuery := h.queryHandler(routePrefix)

	return func(w http.ResponseWriter, r *http.Request) {
		// TODO (b5) - SQL queries are a "read only" thing, but feels like they
		// should have separate configuration to enable when the API is in readonly
		// mode
		if h.ReadOnly {
			readOnlyResponse(w, routePrefix)
			return
		}

		switch r.Method {
		case http.MethodGet, http.MethodPost:
			handleQuery(w, r)
		default:
			util.NotFoundHandler(w, r)
		}
	}
}

func (h *SQLHandlers) queryHandler(routePrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := &lib.SQLQueryParams{
			OutputFormat: "json",
		}

		switch r.Header.Get("Content-Type") {
		case "application/json":
			if err := json.NewDecoder(r.Body).Decode(p); err != nil {
				util.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
		default:
			p.Query = r.FormValue("query")
			if format := r.FormValue("output_format"); format != "" {
				p.OutputFormat = format
			}
		}

		var res []byte
		if err := h.Exec(p, &res); err != nil {
			util.WriteErrResponse(w, http.StatusUnprocessableEntity, err)
			return
		}

		if p.OutputFormat == "json" {
			util.WriteResponse(w, json.RawMessage(res))
			return
		}

		// TODO (b5) - set Content-Type header based on output format
		w.Write(res)
	}
}
