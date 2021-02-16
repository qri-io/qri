package api

import (
	"fmt"
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/dsref"
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
	req := lib.NewLogMethods(inst)
	h := LogHandlers{
		lm: *req,
	}
	return &h
}

// LogHandler is the endpoint for dataset logs
func (h *LogHandlers) LogHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.logHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *LogHandlers) logHandler(w http.ResponseWriter, r *http.Request) {
	args, err := lib.DsRefFromPath(r.URL.Path[len("/history"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if args.Name == "" && args.Path == "" {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("name of dataset or path needed"))
		return
	}

	lp := lib.ListParamsFromRequest(r)
	lp.Peername = args.Username

	local := r.FormValue("local") == "true"
	remoteName := r.FormValue("remote")
	pull := r.FormValue("pull") == "true"

	if local && (remoteName != "" || pull) {
		util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("cannot use the 'local' param with either the 'remote' or 'pull' params"))
		return
	} else if local {
		remoteName = "local"
	}

	res := []dsref.VersionInfo{}
	params := &lib.LogParams{
		Ref:        args.String(),
		Source:     remoteName,
		Pull:       pull,
		ListParams: lp,
	}
	if err := h.lm.Log(params, &res); err != nil {
		if err == repo.ErrNoHistory {
			util.WriteErrResponse(w, http.StatusUnprocessableEntity, err)
			return
		}
		if err != repo.ErrNotFound {
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
		if local && err == repo.ErrNotFound {
			// TODO (b5): This should be 403 not 500
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
	}

	if err := util.WritePageResponse(w, res, r, params.Page()); err != nil {
		log.Infof("error list dataset history response: %s", err.Error())
	}
	return
}
