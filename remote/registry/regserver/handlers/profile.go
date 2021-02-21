package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	apiutil "github.com/qri-io/qri/api/util"
	testPeers "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/remote/registry"
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
				profiles.Create(pro.Username, pro)
			}

			apiutil.WriteResponse(w, ps)
		case "GET":

			l, err := profiles.Len()
			if err != nil {
				apiutil.WriteErrResponse(w, http.StatusInternalServerError, err)
			}
			ps := make([]*registry.Profile, l)

			i := 0
			profiles.SortedRange(func(key string, p *registry.Profile) (bool, error) {
				ps[i] = p
				i++
				return true, nil
			})

			apiutil.WriteResponse(w, ps)
		default:
			apiutil.NotFoundHandler(w, r)
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
			var err error
			if p.Username != "" {
				p, err = profiles.Load(p.Username)
			} else {
				var ok bool
				err = profiles.Range(func(_ string, profile *registry.Profile) (bool, error) {
					if profile.ProfileID == p.ProfileID || profile.PublicKey == p.PublicKey {
						p = profile
						ok = true
						return true, nil
					}
					return false, nil
				})
				if !ok {
					err = registry.ErrNotFound
				}
			}
			if err != nil {
				apiutil.NotFoundHandler(w, r)
				return
			}
		case "POST":
			if err := registry.RegisterProfile(profiles, p); err != nil {
				apiutil.WriteErrResponse(w, http.StatusBadRequest, err)
				return
			}
		case "PUT":
			if err := registry.UpdateProfile(profiles, p); err != nil {
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

// NewProveKeyHandler creates a handler that implements provekey
func NewProveKeyHandler(profiles registry.Profiles) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO(dustmop): Lookup account referred to by the request,
		// and retrieve appropriate data.
		info := testPeers.GetTestPeerInfo(3)
		res := map[string]string{}
		res["profileID"] = info.EncodedPeerID
		apiutil.WriteResponse(w, res)
	}
}
