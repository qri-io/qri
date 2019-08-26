package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/qri-io/apiutil"
	"github.com/qri-io/qri/registry"
)

// NewReputationHandler creates a profile handler func that operates on a *registry.Reputations
func NewReputationHandler(rs registry.Reputations) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rep := &registry.Reputation{}
		switch r.Header.Get("Content-Type") {
		case "application/json":
			if err := json.NewDecoder(r.Body).Decode(rep); err != nil {
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
			// TODO: finesse
			// For now, if no reputation is found, create a new reputation
			// add it to the list of reputations, and return the new reputation
			profileID := rep.ProfileID
			if profileID != "" {
				rep, ok = rs.Load(profileID)
				if !ok {
					rep = registry.NewReputation(profileID)
					rs.Add(rep)
				}
			}
		default:
			apiutil.NotFoundHandler(w, r)
			return
		}
		res := registry.ReputationResponse{
			Reputation: rep,
			Expiration: time.Hour * 24,
		}
		apiutil.WriteResponse(w, res)
	}
}
