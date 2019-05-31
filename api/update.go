package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	util "github.com/qri-io/apiutil"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

// UpdateHandlers wraps lib.UpdateMethods, adding HTTP JSON API handles
type UpdateHandlers struct {
	*lib.UpdateMethods
	ReadOnly bool
}

// UpdatesHandler brings a dataset to the latest version
func (h *UpdateHandlers) UpdatesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.listUpdatesHandler(w, r)
	case "POST":
		h.scheduleUpdateHandler(w, r)
	case "DELETE":
		h.unscheduleUpdateHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h UpdateHandlers) listUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	args := lib.ListParamsFromRequest(r)
	res := []*lib.Job{}
	if err := h.List(&args, &res); err != nil {
		log.Errorf("listing update jobs: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WritePageResponse(w, res, r, args.Page()); err != nil {
		log.Errorf("list jobs response: %s", err.Error())
	}
}

func (h UpdateHandlers) getUpdateHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	res := &lib.Job{}
	if err := h.Job(&name, res); err != nil {
		log.Errorf("getting update job: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WriteResponse(w, res); err != nil {
		log.Errorf("get job response: %s", err.Error())
	}
}

func (h UpdateHandlers) scheduleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var p *lib.ScheduleParams
	res := &lib.Job{}
	switch r.Header.Get("Content-Type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(p); err != nil {
			log.Infof("decoding ScheduleParams: %s", err)
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
	default:
		p = &lib.ScheduleParams{
			Name:        r.FormValue("name"),
			Periodicity: r.FormValue("periodicity"),
			// TODO (b5) - support setting dataset params via form values
			// we should ensure sure pointer is nil if no values are specified
		}
	}

	if err := h.Schedule(p, res); err != nil {
		// TODO (b5): error returned here could be either a "BadRequest"
		// or an Internal Error. disambiguate.
		util.WriteErrResponse(w, http.StatusBadRequest, err)
	}
}

func (h UpdateHandlers) unscheduleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	res := false
	if err := h.Unschedule(&name, &res); err != nil {
		log.Errorf("decoding ScheduleParams: %s", err)
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}
}

// LogsHandler shows the log of previously run updates
func (h *UpdateHandlers) LogsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		h.logsHandler(w, r)
	default:
		util.NotFoundHandler(w, r)
	}
}

func (h *UpdateHandlers) logsHandler(w http.ResponseWriter, r *http.Request) {
	args := lib.ListParamsFromRequest(r)
	res := []*lib.Job{}
	if err := h.Logs(&args, &res); err != nil {
		log.Errorf("listing update logs: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := util.WritePageResponse(w, res, r, args.Page()); err != nil {
		log.Errorf("list jobs response: %s", err.Error())
	}
}

// LogFileHandler fetches log output file data
func (h *UpdateHandlers) LogFileHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("log_name")
	data := []byte{}
	if err := h.LogFile(&name, &data); err != nil {
		log.Errorf("getting update log file: %s", err.Error())
		util.WriteErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// RunHandler brings a dataset to the latest version
func (h UpdateHandlers) RunHandler(w http.ResponseWriter, r *http.Request) {
	if h.ReadOnly || r.Method != "POST" {
		util.NotFoundHandler(w, r)
		return
	}

	ref, err := DatasetRefFromPath(r.URL.Path[len("/update/run"):])
	if err != nil {
		util.WriteErrResponse(w, http.StatusBadRequest, err)
		return
	}

	p := &lib.SaveParams{
		Ref:        ref.String(),
		Title:      r.FormValue("title"),
		Message:    r.FormValue("message"),
		DryRun:     r.FormValue("dry_run") == "true",
		ReturnBody: false,
	}

	if r.FormValue("secrets") != "" {
		p.Secrets = map[string]string{}
		if err := json.Unmarshal([]byte(r.FormValue("secrets")), &p.Secrets); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, fmt.Errorf("parsing secrets: %s", err))
			return
		}
	}

	res := &repo.DatasetRef{}
	// TODO (b5) - finish
	// if err := h.DatasetRequests.Update(p, res); err != nil {
	// 	util.WriteErrResponse(w, http.StatusInternalServerError, err)
	// 	return
	// }
	util.WriteResponse(w, res)
}

// ServiceHandler configures & reports on the update daemon
func (h UpdateHandlers) ServiceHandler(w http.ResponseWriter, r *http.Request) {
	if h.ReadOnly {
		util.NotFoundHandler(w, r)
		return
	}

	in := false
	switch r.Method {
	case "OPTIONS":
		util.EmptyOkHandler(w, r)
	case "GET":
		res := &lib.ServiceStatus{}
		if err := h.ServiceStatus(&in, res); err != nil {
			log.Errorf("getting service status: %s", err)
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
	case "POST":
		daemonize, _ := util.ReqParamBool("daemonize", r)
		var res bool
		p := &lib.UpdateServiceStartParams{
			Ctx:       context.Background(),
			Daemonize: daemonize,
		}
		if err := h.ServiceStart(p, &res); err != nil {
			log.Errorf("starting service: %s", err)
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
	case "DELETE":
		res := false
		if err := h.ServiceStop(&in, &res); err != nil {
			log.Errorf("stopping service: %s", err)
			util.WriteErrResponse(w, http.StatusInternalServerError, err)
			return
		}
	default:
		util.NotFoundHandler(w, r)
	}

}
