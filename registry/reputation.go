package registry

import (
	"fmt"
	"time"
)

// Reputation is record of the peers reputation on the network
// TODO: this is a stub that can and should be expanded
type Reputation struct {
	ProfileID string
	Rep       int
}

// NewReputation creates a new reputation. Reputations start at 1 for now
// REPUTATIONS MUST BE NON-ZERO NUMBERS
func NewReputation(id string) *Reputation {
	return &Reputation{
		ProfileID: id,
		Rep:       1,
	}
}

// Validate is a sanity check that all required values are present
func (r *Reputation) Validate() error {
	if r.ProfileID == "" {
		return fmt.Errorf("profileID is required")
	}

	return nil
}

// SetReputation sets the reputation of a given Reputation
func (r *Reputation) SetReputation(reputation int) {
	r.Rep = reputation
}

// Reputation gets the rep of a given Reputation
func (r *Reputation) Reputation() int {
	return r.Rep
}

// ReputationResponse is the result of a request for a reputation
type ReputationResponse struct {
	Reputation *Reputation
	Expiration time.Duration
}
