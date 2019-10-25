// Package profile defines a qri peer profile
package profile

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/config"

	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

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
	Thumb string `json:"thumb"`
	// Profile photo
	Photo string `json:"photo"`
	// Poster photo for users's profile page
	Poster string `json:"poster"`
	// Twitter is a  peer's twitter handle
	Twitter string `json:"twitter"`
	// Online indicates if this peer is currently connected to the network
	Online bool `json:"online,omitempty"`
	// PeerIDs lists any network PeerIDs associated with this profile
	// in the form /network/peerID
	PeerIDs []peer.ID `json:"peerIDs"`
	// NetworkAddrs keeps a list of locations for this profile on the network as multiaddr strings
	NetworkAddrs []ma.Multiaddr `json:"networkAddrs,omitempty"`
}

// NewProfile allocates a profile from a CodingProfile
func NewProfile(p *config.ProfilePod) (pro *Profile, err error) {
	pro = &Profile{}
	err = pro.Decode(p)
	return
}

// Decode turns a ProfilePod into a profile.Profile
func (p *Profile) Decode(sp *config.ProfilePod) error {
	id, err := IDB58Decode(sp.ID)
	if err != nil {
		return fmt.Errorf("parsing profile.ID \"%s\": %s", sp.ID, err)
	}

	t, err := ParseType(sp.Type)
	if err != nil {
		return fmt.Errorf("parsing profileType \"%s\": %s", sp.Type, err)
	}

	pids := make([]peer.ID, len(sp.PeerIDs))
	for i, idstr := range sp.PeerIDs {
		idstr = strings.TrimPrefix(idstr, "/ipfs/")
		if id, err := peer.IDB58Decode(idstr); err == nil {
			pids[i] = id
		}
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
		PeerIDs:     pids,
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
		pro.Thumb = sp.Thumb
	}

	if sp.Poster != "" {
		pro.Poster = sp.Poster
	}

	if sp.Photo != "" {
		pro.Photo = sp.Photo
	}

	for _, addrStr := range sp.NetworkAddrs {
		if maddr, err := ma.NewMultiaddr(addrStr); err == nil {
			pro.NetworkAddrs = append(pro.NetworkAddrs, maddr)
		}
	}

	*p = pro
	return nil
}

// Encode returns a ProfilePod for a given profile
func (p Profile) Encode() (*config.ProfilePod, error) {
	pids := make([]string, len(p.PeerIDs))
	for i, pid := range p.PeerIDs {
		pids[i] = fmt.Sprintf("/ipfs/%s", pid.Pretty())
	}
	var addrs []string
	for _, maddr := range p.NetworkAddrs {
		addrs = append(addrs, maddr.String())
	}
	pp := &config.ProfilePod{
		ID:           p.ID.String(),
		Type:         p.Type.String(),
		Peername:     p.Peername,
		Created:      p.Created,
		Updated:      p.Updated,
		Email:        p.Email,
		Name:         p.Name,
		Description:  p.Description,
		HomeURL:      p.HomeURL,
		Color:        p.Color,
		Twitter:      p.Twitter,
		Poster:       p.Poster,
		Photo:        p.Photo,
		Thumb:        p.Thumb,
		Online:       p.Online,
		PeerIDs:      pids,
		NetworkAddrs: addrs,
	}
	return pp, nil
}
