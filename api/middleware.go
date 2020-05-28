package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	util "github.com/qri-io/apiutil"
)

// middleware handles request logging
func (s Server) middleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Infof("%s %s %s", r.Method, r.URL.Path, time.Now())

		// TODO - Strict Transport config?
		// if cfg.TLS {
		// 	// If TLS is enabled, set 1 week strict TLS, 1 week for now to prevent catastrophic mess-ups
		// 	w.Header().Add("Strict-Transport-Security", "max-age=604800")
		// }
		s.addCORSHeaders(w, r)

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

func (s *Server) datasetRefMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ref, _ := DatasetRefFromReq(r); !ref.IsEmpty() {
			ctx := context.WithValue(r.Context(), DatasetRefCtxKey, ref)
			r = r.WithContext(ctx)
		}
		handler(w, r)
	}
}
