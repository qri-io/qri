package profile

import (
	"time"

	"github.com/ipfs/go-datastore"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// Profile defines peer profile details
type Profile struct {
	ID string `json:"id"`
	// Created timestamp
	Created time.Time `json:"created,omitempty"`
	// Updated timestamp
	Updated time.Time `json:"updated,omitempty"`
	// handle for the user. min 1 character, max 80. composed of [_,-,a-z,A-Z,1-9]
	Username string `json:"username"`
	// specifies weather this is a user or an organization
	Type UserType `json:"type"`
	// user's email address
	Email string `json:"email"`
	// user name field. could be first[space]last, but not strictly enforced
	Name string `json:"name"`
	// user-filled description of self
	Description string `json:"description"`
	// url this user wants the world to click
	HomeURL string `json:"homeUrl"`
	// color this user likes to use as their theme color
	Color string `json:"color"`
	// Thumb path for user's thumbnail
	Thumb datastore.Key `json:"thumb"`
	// Profile photo
	Profile datastore.Key `json:"profile"`
	// Poster photo for users's profile page
	Poster datastore.Key `json:"poster"`
	// users's twitter handle
	Twitter string `json:"twitter"`
}

// PeerID gives a peer.ID for this profile
func (p *Profile) PeerID() (peer.ID, error) {
	return peer.IDB58Decode(p.ID)
}
