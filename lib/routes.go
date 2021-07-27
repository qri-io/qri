package lib

import (
	"net/http"

	"github.com/gorilla/mux"
	qhttp "github.com/qri-io/qri/lib/http"
)

// GiveAPIServer creates an API server that gives access to lib's registered methods
func (inst *Instance) GiveAPIServer(middleware func(handler http.HandlerFunc) http.HandlerFunc, ignoreMethods []string) *mux.Router {
	m := mux.NewRouter()
	for methodName, call := range inst.regMethods.reg {
		if arrayContainsString(ignoreMethods, methodName) {
			continue
		}
		if call.Endpoint == qhttp.DenyHTTP {
			continue
		}
		handler := middleware(NewHTTPRequestHandler(inst, methodName))
		// All endpoints use POST verb
		httpVerb := http.MethodPost
		m.Handle(string(call.Endpoint), handler).Methods(httpVerb, http.MethodOptions)
	}
	return m
}

func arrayContainsString(searchSpace []string, target string) bool {
	for _, elem := range searchSpace {
		if elem == target {
			return true
		}
	}
	return false
}
