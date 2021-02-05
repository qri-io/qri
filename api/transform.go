package api

import (
	"encoding/json"
	"net/http"

	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/lib"
)

// TransformHandlers connects HTTP requests to the TransformMethods subsystem
type TransformHandlers struct {
	*lib.TransformMethods
}

// NewTransformHandlers constructs a TrasnformHandlers struct
func NewTransformHandlers(inst *lib.Instance) TransformHandlers {
	return TransformHandlers{TransformMethods: lib.NewTransformMethods(inst)}
}

// ApplyHandler is an HTTP handler function for executing a transform script
func (h TransformHandlers) ApplyHandler(prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := lib.ApplyParams{}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		res, err := h.TransformMethods.Apply(r.Context(), &p)
		if err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		util.WriteResponse(w, res)
	}
}
