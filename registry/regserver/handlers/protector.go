package handlers

import (
	"errors"
	"net/http"

	"github.com/qri-io/apiutil"
)

// MethodProtector is an interface for controling access to http.HandlerFunc's
// according to request method (GET, PUT, POST, etc.)
type MethodProtector interface {
	// ProtectMethods should accept a list of http request methods and return a function that
	// generates middleware to screen for authorization on those methods.
	//
	// For example if "reqHandler" is an http.HandlerFunc, the following would check for
	// authentication on all POST requests:
	//  Protector.ProtectMethods("POST")(reqHandler)
	ProtectMethods(methods ...string) func(h http.HandlerFunc) http.HandlerFunc
}

// BAProtector implements HTTP Basic Auth checking as a protector
type BAProtector struct {
	username, password string
}

// NewBAProtector creates a HTTP basic auth protector from a username/password combo
func NewBAProtector(username, password string) BAProtector {
	return BAProtector{username, password}
}

// ProtectMethods implements the MethodProtector interface
func (ba BAProtector) ProtectMethods(methods ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			for _, m := range methods {
				if r.Method == m || m == "*" {
					username, password, set := r.BasicAuth()
					if !set || username != ba.username || password != ba.password {
						log.Infof("invalid key")
						apiutil.WriteErrResponse(w, http.StatusForbidden, errors.New("invalid key"))
						return
					}
				}
			}

			h.ServeHTTP(w, r)
		}
	}
}

// NoopProtector implements the MethodProtector without doing any checks
type NoopProtector uint8

// NewNoopProtector creates a NoopProtector
func NewNoopProtector() NoopProtector {
	return NoopProtector(0)
}

// ProtectMethods implements the MethodProtector interface
func (NoopProtector) ProtectMethods(methods ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
}
