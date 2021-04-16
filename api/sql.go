package api

import (
	"encoding/json"
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
)

// SQLHandlers connects HTTP requests to the FSI subsystem
type SQLHandlers struct {
	Instance *lib.Instance
	ReadOnly bool
}

// NewSQLHandlers creates handlers that talk to qri's filesystem integration
func NewSQLHandlers(inst *lib.Instance, readOnly bool) SQLHandlers {
	return SQLHandlers{
		Instance: inst,
		ReadOnly: readOnly,
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

	got, _, err := h.Instance.Dispatch(r.Context(), "sql.exec", params)
	if err != nil {
		util.RespondWithError(w, err)
		return
	}
	res, ok := got.([]byte)
	if !ok {
		util.RespondWithDispatchTypeError(w, got)
		return
	}
	h.replyWithSQLResponse(w, r, params, res)
}

func (h *SQLHandlers) replyWithSQLResponse(w http.ResponseWriter, r *http.Request, p *lib.SQLQueryParams, res []byte) {
	if p.Format == "json" {
		util.WriteResponse(w, json.RawMessage(res))
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(res)
		if err != nil {
			log.Debugf("failed writing results of SQL query")
		}
	}
}
