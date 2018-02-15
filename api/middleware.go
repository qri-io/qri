package api

import (
	"context"
	"net/http"
	"time"
)

// middleware handles request logging
func (s *Server) middleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.log.Infof("%s %s %s\n", r.Method, r.URL.Path, time.Now())

		// If this server is operating behind a proxy, but we still want to force
		// users to use https, cfg.ProxyForceHttps == true will listen for the common
		// X-Forward-Proto & redirect to https
		if s.cfg.ProxyForceHTTPS {
			if r.Header.Get("X-Forwarded-Proto") == "http" {
				w.Header().Set("Connection", "close")
				url := "https://" + r.Host + r.URL.String()
				http.Redirect(w, r, url, http.StatusMovedPermanently)
				return
			}
		}

		// TODO - Strict Transport config?
		// if cfg.TLS {
		// 	// If TLS is enabled, set 1 week strict TLS, 1 week for now to prevent catastrophic mess-ups
		// 	w.Header().Add("Strict-Transport-Security", "max-age=604800")
		// }
		s.addCORSHeaders(w, r)

		handler(w, r)
	}
}

// addCORSHeaders adds CORS header info for whitelisted servers
func (s *Server) addCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	for _, o := range s.cfg.AllowedOrigins {
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
