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
func (h *SQLHandlers) QueryHandler(w http.ResponseWriter, r *http.Request) {
	// TODO (b5) - SQL queries are a "read only" thing, but feels like they
	// should have separate configuration to enable when the API is in readonly
	// mode
	if h.ReadOnly {
		readOnlyResponse(w, "/sql")
		return
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.queryHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *SQLHandlers) queryHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.SQLQueryParams{}

	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.Exec(r.Context(), params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusUnprocessableEntity, err)
		return
	}

	if params.OutputFormat == "json" {
		util.WriteResponse(w, json.RawMessage(res))
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(res)
		if err != nil {
			log.Debugf("failed writing results of SQL query")
		}
	}
}
