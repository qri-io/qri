package profile

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/qri/config"
	// ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

// CodingProfile is a version of Profile that's specifically intended for
// serialization (encoding & decoding)
type CodingProfile struct {
	config.Profile
	Addresses map[string][]string `json:"addresses,omitempty"`
}

// NewCodingProfile creates a serializeprofile from a config.Profile
func NewCodingProfile(pro *config.Profile) *CodingProfile {
	return &CodingProfile{
		Profile:   *pro,
		Addresses: map[string][]string{},
	}
}

// Profile defines peer profile details
type Profile struct {
	ID ID `json:"id"`
	// Created timestamp
	Created time.Time `json:"created,omitempty"`
	// Updated timestamp
	Updated time.Time `json:"updated,omitempty"`
	// PrivKey is the peer's private key, should only be present for the current peer
	PrivKey crypto.PrivKey `json:"_,omitempty"`
	// Peername a handle for the user. min 1 character, max 80. composed of [_,-,a-z,A-Z,1-9]
	Peername string `json:"peername"`
	// specifies weather this is a user or an organization
	Type Type `json:"type"`
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
	Photo datastore.Key `json:"photo"`
	// Poster photo for users's profile page
	Poster datastore.Key `json:"poster"`
	// Twitter is a  peer's twitter handle
	Twitter string `json:"twitter"`
	// Addresses lists any network addresses associated with this profile
	// in the form of peer.ID.Pretty() : []multiaddr strings
	// both peer.IDs and multiaddresses are converted to strings for
	// clean en/decoding
	Addresses map[string][]string `json:"addresses"`
}

// NewProfile allocates a profile from a CodingProfile
func NewProfile(p *config.Profile) (pro *Profile, err error) {
	pro = &Profile{}
	err = pro.Decode(NewCodingProfile(p))
	return
}

// PeerIDs sifts through listed multaddrs looking for an IPFS peer ID
func (p *Profile) PeerIDs() (ids []peer.ID) {
	for idstr := range p.Addresses {
		if id, err := peer.IDB58Decode(idstr); err == nil {
			ids = append(ids, id)
		}
	}
	return
}

// Decode turns a CodingProfile into a profile.Profile
func (p *Profile) Decode(sp *CodingProfile) error {
	id, err := IDB58Decode(sp.ID)
	if err != nil {
		return err
	}

	t, err := ParseType(sp.Type)
	if err != nil {
		return err
	}

	pro := Profile{
		ID:          id,
		Type:        t,
		Peername:    sp.Peername,
		Created:     sp.Created,
		Updated:     sp.Updated,
		Email:       sp.Email,
		Name:        sp.Name,
		Description: sp.Description,
		HomeURL:     sp.HomeURL,
		Color:       sp.Color,
		Twitter:     sp.Twitter,
		Addresses:   sp.Addresses,
	}

	if sp.PrivKey != "" {
		data, err := base64.StdEncoding.DecodeString(sp.PrivKey)
		if err != nil {
			return fmt.Errorf("decoding private key: %s", err.Error())
		}

		pk, err := crypto.UnmarshalPrivateKey(data)
		if err != nil {
			return fmt.Errorf("invalid private key: %s", err.Error())
		}
		pro.PrivKey = pk
	}

	if sp.Thumb != "" {
		pro.Thumb = datastore.NewKey(sp.Thumb)
	}

	if sp.Poster != "" {
		pro.Poster = datastore.NewKey(sp.Poster)
	}

	if sp.Photo != "" {
		pro.Photo = datastore.NewKey(sp.Photo)
	}

	*p = pro
	return nil
}

// Encode returns a CodingProfile for a given profile
func (p Profile) Encode() (*CodingProfile, error) {
	sp := &CodingProfile{
		Profile: config.Profile{
			ID:          p.ID.String(),
			Type:        p.Type.String(),
			Peername:    p.Peername,
			Created:     p.Created,
			Updated:     p.Updated,
			Email:       p.Email,
			Name:        p.Name,
			Description: p.Description,
			HomeURL:     p.HomeURL,
			Color:       p.Color,
			Twitter:     p.Twitter,
			Poster:      p.Poster.String(),
			Photo:       p.Photo.String(),
			Thumb:       p.Thumb.String(),
		},
		Addresses: p.Addresses,
	}
	return sp, nil
}
