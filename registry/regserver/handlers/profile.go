package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/qri-io/apiutil"
	"github.com/qri-io/qri/registry"
)

// NewProfilesHandler creates a profiles handler function that operates
// on a *registry.Profiles
func NewProfilesHandler(profiles registry.Profiles) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			ps := []*registry.Profile{}
			switch r.Header.Get("Content-Type") {
			case "application/json":
				if err := json.NewDecoder(r.Body).Decode(&ps); err != nil {
					apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
					return
				}
			default:
				err := fmt.Errorf("Content-Type must be application/json")
				apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}

			for _, pro := range ps {
				profiles.Store(pro.Handle, pro)
			}
			fallthrough
		case "GET":
			ps := make([]*registry.Profile, profiles.Len())

			i := 0
			profiles.SortedRange(func(key string, p *registry.Profile) bool {
				ps[i] = p
				i++
				return false
			})

			apiutil.WriteResponse(w, ps)
		}
	}
}

// NewProfileHandler creates a profile handler func that operats on
// a *registry.Profiles
func NewProfileHandler(profiles registry.Profiles) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := &registry.Profile{}
		switch r.Header.Get("Content-Type") {
		case "application/json":
			if err := json.NewDecoder(r.Body).Decode(p); err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
		default:
			err := fmt.Errorf("Content-Type must be application/json")
			apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
			return
		}

		switch r.Method {
		case "GET":
			var ok bool
			if p.Handle != "" {
				p, ok = profiles.Load(p.Handle)
			} else {
				profiles.Range(func(handle string, profile *registry.Profile) bool {
					if profile.ProfileID == p.ProfileID || profile.PublicKey == p.PublicKey {
						p = profile
						ok = true
						return true
					}
					return false
				})
			}
			if !ok {
				apiutil.NotFoundHandler(w, r)
				return
			}
		case "PUT", "POST":
			if err := registry.RegisterProfile(profiles, p); err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
		case "DELETE":
			if err := registry.DeregisterProfile(profiles, p); err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
		default:
			apiutil.NotFoundHandler(w, r)
			return
		}

		apiutil.WriteResponse(w, p)
	}
}
