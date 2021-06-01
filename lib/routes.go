package lib

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/qri-io/qri/auth/token"
)

// GiveAPIServer creates an API server that gives access to lib's registered methods
func (inst *Instance) GiveAPIServer(middleware func(handler http.HandlerFunc) http.HandlerFunc, ignoreMethods []string) *mux.Router {
	m := mux.NewRouter()
	for methodName, call := range inst.regMethods.reg {
		if arrayContainsString(ignoreMethods, methodName) {
			continue
		}
		if call.Endpoint == DenyHTTP {
			continue
		}
		handler := middleware(NewHTTPRequestHandler(inst, methodName))
		// All endpoints use POST verb
		httpVerb := http.MethodPost
		m.Handle(string(call.Endpoint), handler).Methods(httpVerb, http.MethodOptions)
	}

	m.Handle("/oauth/token", middleware(func(w http.ResponseWriter, r *http.Request) {
		own := inst.profiles.Owner()
		tok, err := token.NewPrivKeyAuthToken(own.PrivKey, own.ID.String(), time.Hour*24*14)
		if err != nil {
			w.Write([]byte(fmt.Sprintf("creating token: %s", err)))
			return
		}

		res := map[string]interface{}{
			"meta": map[string]interface{}{
				"code": 200,
			},
			"data": map[string]interface{}{
				"access_token": tok,
			},
		}

		json.NewEncoder(w).Encode(res)
	}))

	m.Handle("/identity/session", middleware(func(w http.ResponseWriter, r *http.Request) {
		own := inst.profiles.Owner()
		res := map[string]interface{}{
			"meta": map[string]interface{}{
				"code": 200,
			},
			"data": own,
		}

		json.NewEncoder(w).Encode(res)
	}))

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
