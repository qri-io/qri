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
	}
}

// corsMiddleware adds Cross-Origin Resource Sharing headers for any request
// who's origin matches one of allowedOrigins
func corsMiddleware(allowedOrigins []string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			for _, o := range allowedOrigins {
				if origin == o {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			// intercept OPTIONS requests with an early return
			if r.Method == http.MethodOptions {
				util.EmptyOkHandler(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// muxVarsToQueryParamMiddleware moves all mux variables to query parameter
// values, failing with an error if a name collision with user-provided query
// params occurs
func muxVarsToQueryParamMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := setMuxVarsToQueryParams(r); err != nil {
			util.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func setMuxVarsToQueryParams(r *http.Request) error {
	q := r.URL.Query()
	for varName, val := range mux.Vars(r) {
		if q.Get(varName) != "" {
			return fmt.Errorf("conflict in query param: %s = %s", varName, val)
		}
		q.Add(varName, val)
	}
	r.URL.RawQuery = q.Encode()
	return nil
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
		Username: mvars["username"],
		Name:     mvars["name"],
		Path:     muxVarsPath(mvars),
	}

	if refstr := ref.String(); refstr != "" {
		q := r.URL.Query()
		q.Add("ref", refstr)
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
