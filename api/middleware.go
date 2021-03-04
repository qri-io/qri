package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/qri-io/qri/api/util"
	"github.com/qri-io/qri/dsref"
)

// Middleware handles request logging
func (s Server) Middleware(handler http.HandlerFunc) http.HandlerFunc {
	return s.mwFunc(handler, true)
}

// NoLogMiddleware runs middleware without logging the request
func (s Server) NoLogMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return s.mwFunc(handler, false)
}

func (s Server) mwFunc(handler http.HandlerFunc, shouldLog bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if shouldLog {
			log.Infof("%s %s %s", r.Method, r.URL.Path, time.Now())
		}

		s.addCORSHeaders(w, r)
		if r.Method == http.MethodOptions {
			util.EmptyOkHandler(w, r)
			return
		}

		if ok := s.readOnlyCheck(r); ok {
			handler(w, r)
		} else {
			util.WriteErrResponse(w, http.StatusForbidden, fmt.Errorf("qri server is in read-only mode, only certain GET requests are allowed"))
		}
	}
}

func (s *Server) readOnlyCheck(r *http.Request) bool {
	return !s.Config().API.ReadOnly || r.Method == "GET" || r.Method == "OPTIONS"
}

// addCORSHeaders adds CORS header info for whitelisted servers
func (s *Server) addCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	for _, o := range s.Config().API.AllowedOrigins {
		if origin == o {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			return
		}
	}
}

// refStringMiddleware converts gorilla mux params to a "refstr" query parmeter
// and adds it to an http request
func refStringMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setRefStringFromMuxVars(r)
		next.ServeHTTP(w, r)
	})
}

func setRefStringFromMuxVars(r *http.Request) {
	mvars := mux.Vars(r)
	ref := dsref.Ref{
		Username: mvars["peername"],
		Name:     mvars["name"],
		Path:     muxVarsPath(mvars),
	}

	if refstr := ref.String(); refstr != "" {
		q := r.URL.Query()
		q.Add("refstr", refstr)
		r.URL.RawQuery = q.Encode()
	}
}

func muxVarsPath(mvars map[string]string) string {
	fs := mvars["fs"]
	hash := mvars["hash"]
	if fs != "" && hash != "" {
		return fmt.Sprintf("/%s/%s", fs, hash)
	}
	return ""
}

func stripServerSideQueryParams(r *http.Request) {
	q := r.URL.Query()
	q.Del("refstr")
	r.URL.RawQuery = q.Encode()
}
