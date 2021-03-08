package api

import (
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
)

// TransformHandlers connects HTTP requests to the TransformMethods subsystem
type TransformHandlers struct {
	inst *lib.Instance
}

// NewTransformHandlers constructs a TrasnformHandlers struct
func NewTransformHandlers(inst *lib.Instance) TransformHandlers {
	return TransformHandlers{inst: inst}
}

// ApplyHandler is an HTTP handler function for executing a transform script
func (h TransformHandlers) ApplyHandler(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		method := "transform.apply"
		p := h.inst.NewInputParam(method)

		if err := lib.UnmarshalParams(r, p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		res, _, err := h.inst.Dispatch(r.Context(), method, p)
		if err != nil {
			util.RespondWithError(w, err)
			return
		}
		util.WriteResponse(w, res)
		return
	}
}
