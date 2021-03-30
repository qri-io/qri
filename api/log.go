package api

import (
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

// LogHandlers wraps a LogMethods with http.HandlerFuncs
type LogHandlers struct {
	lm       lib.LogMethods
	readOnly bool
}

// NewLogHandlers allocates a LogHandlers pointer
func NewLogHandlers(inst *lib.Instance) *LogHandlers {
	h := LogHandlers{
		lm: inst.Log(),
	}
	return &h
}

// LogHandler is the endpoint for dataset logs
func (h *LogHandlers) LogHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.logHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// LogbookHandler is the endpoint for a dataset logbook
func (h *LogHandlers) LogbookHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.logbookHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// PlainLogsHandler is the endpoint for getting the full logbook in human readable form
func (h *LogHandlers) PlainLogsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.plainLogsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

// LogbookSummaryHandler is the endpoint for a string overview of the logbook
func (h *LogHandlers) LogbookSummaryHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		h.logbookSummaryHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *LogHandlers) logHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.LogParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.lm.Log(r.Context(), params)
	if err != nil {
		if err == repo.ErrNoHistory {
			util.WriteErrResponse(w, http.StatusUnprocessableEntity, err)
			return
		}
		if err != repo.ErrNotFound {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		if params.Source == "local" && err == repo.ErrNotFound {
			util.WriteErrResponse(w, http.StatusForbidden, err)
			return
		}
	}

	if err := util.WritePageResponse(w, res, r, params.Page()); err != nil {
		log.Infof("error list dataset history response: %s", err.Error())
	}
}

func (h *LogHandlers) logbookHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.RefListParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.lm.Logbook(r.Context(), params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WritePageResponse(w, res, r, util.Page{})
}

func (h *LogHandlers) plainLogsHandler(w http.ResponseWriter, r *http.Request) {
	params := &lib.PlainLogsParams{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.lm.PlainLogs(r.Context(), params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WritePageResponse(w, res, r, util.Page{})
}

func (h *LogHandlers) logbookSummaryHandler(w http.ResponseWriter, r *http.Request) {
	params := &struct{}{}
	err := lib.UnmarshalParams(r, params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	res, err := h.lm.LogbookSummary(r.Context(), params)
	if err != nil {
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	util.WritePageResponse(w, res, r, util.Page{})
}
