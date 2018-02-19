package core

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/rpc"
	"os"
	"time"

	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/jsonschema"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// ProfileRequests encapsulates business logic for this node's
// user profile
type ProfileRequests struct {
	repo repo.Repo
	cli  *rpc.Client
}

// CoreRequestsName implements the Requets interface
func (ProfileRequests) CoreRequestsName() string { return "profile" }

// NewProfileRequests creates a ProfileRequests pointer from either a repo
// or an rpc.Client
func NewProfileRequests(r repo.Repo, cli *rpc.Client) *ProfileRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewProfileRequests"))
	}

	return &ProfileRequests{
		repo: r,
		cli:  cli,
	}
}

// Profile is a public, gob-encodable version of repo/profile/Profile
type Profile struct {
	ID          string           `json:"id"`
	Created     time.Time        `json:"created,omitempty"`
	Updated     time.Time        `json:"updated,omitempty"`
	Peername    string           `json:"peername"`
	Type        profile.UserType `json:"type"`
	Email       string           `json:"email"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	HomeURL     string           `json:"homeUrl"`
	Color       string           `json:"color"`
	Thumb       string           `json:"thumb"`
	Profile     string           `json:"profile"`
	Poster      string           `json:"poster"`
	Twitter     string           `json:"twitter"`
}

// unmarshalProfile gets a profile.Profile from a Profile pointer
func unmarshalProfile(p *Profile) (*profile.Profile, error) {
	// silly workaround for gob encoding
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("err re-encoding json: %s", err.Error())
	}

	_p := &profile.Profile{}
	if err := json.Unmarshal(data, _p); err != nil {
		return nil, fmt.Errorf("error unmarshaling json: %s", err.Error())
	}

	return _p, nil
}

// marshalProfile gets a Profile pointer from a profile.Profile pointer
func marshalProfile(p *profile.Profile) (*Profile, error) {
	// silly workaround for gob encoding
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("err re-encoding json: %s", err.Error())
	}

	_p := &Profile{}
	if err := json.Unmarshal(data, _p); err != nil {
		return nil, fmt.Errorf("error unmarshaling json: %s", err.Error())
	}

	return _p, nil
}

// GetProfile get's this node's peer profile
func (r *ProfileRequests) GetProfile(in *bool, res *Profile) error {
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.GetProfile", in, res)
	}

	profile, err := r.repo.Profile()
	if err != nil {
		return err
	}

	_p, err := marshalProfile(profile)
	if err != nil {
		return err
	}
	*res = *_p
	return nil
}

// AssignEditable - help with this name plz
// don't really want to call it assign cause that implies it would change every field
// AssignEditable collapses all the editable properties of a Profile onto one.
// this is directly inspired by Javascript's Object.assign
// Editable Fields:
// Peername
// Email
// Name
// Description
// HomeURL
// Color
// Thumb
// Profile
// Poster
// Twitter
func (p *Profile) AssignEditable(profiles ...*Profile) {
	for _, pf := range profiles {
		if pf == nil {
			continue
		}
		if pf.Peername != "" {
			p.Peername = pf.Peername
		}
		if pf.Email != "" {
			p.Email = pf.Email
		}
		if pf.Name != "" {
			p.Name = pf.Name
		}
		if pf.Description != "" {
			p.Description = pf.Description
		}
		if pf.HomeURL != "" {
			p.HomeURL = pf.HomeURL
		}
		if pf.Color != "" {
			p.Color = pf.Color
		}
		if pf.Thumb != "" {
			p.Thumb = pf.Thumb
		}
		if pf.Profile != "" {
			p.Profile = pf.Profile
		}
		if pf.Poster != "" {
			p.Poster = pf.Poster
		}
		if pf.Twitter != "" {
			p.Twitter = pf.Twitter
		}
	}
}

// ValidateProfile validates all fields of profile
// returning first error found
func (p *Profile) ValidateProfile() error {
	// read profileSchema from testdata/profileSchema.json
	// marshal to json?
	gp := os.Getenv("GOPATH")
	filepath := gp + "/src/github.com/qri-io/qri/core/schemas/profileSchema.json"
	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("error opening schemas/profileSchema.json: %s", err)
	}
	defer f.Close()

	profileSchema, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("error reading profileSchema: %s", err)
	}

	// unmarshal to rootSchema
	rs := &jsonschema.RootSchema{}
	if err := json.Unmarshal(profileSchema, rs); err != nil {
		return fmt.Errorf("error unmarshaling profileSchema to RootSchema: %s", err)
	}
	profile, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("error marshaling profile to json: %s", err)
	}
	if errors := rs.ValidateBytes(profile); len(errors) > 0 {
		return fmt.Errorf("%s", errors)
	}
	return nil
}

// SavePeername updates the profile to include the peername
// warning! SHOULD ONLY BE CALLED ONCE: WHEN INITIALIZING/SETTING UP THE REPO
func (r *ProfileRequests) SavePeername(p *Profile, res *Profile) error {
	if p == nil {
		return fmt.Errorf("profile required to save peername")
	}
	if p.Peername == "" {
		return fmt.Errorf("peername required")
	}

	// ensure only fields we want to save are being passed to saveProfile:
	pro := &Profile{
		Peername: p.Peername,
	}

	if err := r.saveProfile(pro, res); err != nil {
		return fmt.Errorf("error saving profile: %s", err)
	}

	return nil
}

// SaveProfile stores changes to this peer's editable profile profile
func (r *ProfileRequests) SaveProfile(p *Profile, res *Profile) error {
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.SaveProfile", p, res)
	}
	if p == nil {
		return fmt.Errorf("profile required for update")
	}

	// to ensure only the fields we want to be editable are passed to saveProfile:
	pro := &Profile{
		Name:        p.Name,
		Email:       p.Email,
		Description: p.Description,
		HomeURL:     p.HomeURL,
		Color:       p.Color,
		Twitter:     p.Twitter,
	}

	if err := r.saveProfile(pro, res); err != nil {
		return fmt.Errorf("error saving profile: %s", err)
	}

	return nil
}

// FileParams defines parameters for Files as arguments to core methods
type FileParams struct {
	// Url      string    // url to download data from. either Url or Data is required
	Filename string    // filename of data file. extension is used for filetype detection
	Data     io.Reader // reader of structured data. either Url or Data is required
}

// SetProfilePhoto changes this peer's profile image
func (r *ProfileRequests) SetProfilePhoto(p *FileParams, res *Profile) error {
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.SetProfilePhoto", p, res)
	}

	if p.Data == nil {
		return fmt.Errorf("file is required")
	}

	data, err := ioutil.ReadAll(p.Data)
	if err != nil {
		return fmt.Errorf("error reading file data: %s", err.Error())
	}

	mimetype := http.DetectContentType(data)
	if mimetype != "image/jpeg" && mimetype != "image/png" {
		return fmt.Errorf("invalid file format. only .jpg & .png images allowed")
	}

	if len(data) > 250000 {
		return fmt.Errorf("file size too large. max size is 250kb")
	}

	path, err := r.repo.Store().Put(memfs.NewMemfileBytes(p.Filename, data), true)
	if err != nil {
		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	pro := &Profile{
		Profile: path.String(),
		// TODO - resize photo
		Thumb: path.String(),
	}
	if err := r.saveProfile(pro, res); err != nil {
		return fmt.Errorf("error saving profile: %s", err)
	}

	return nil
}

// SetPosterPhoto changes this peer's poster image
func (r *ProfileRequests) SetPosterPhoto(p *FileParams, res *Profile) error {
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.SetPosterPhoto", p, res)
	}

	if p.Data == nil {
		return fmt.Errorf("file is required")
	}

	data, err := ioutil.ReadAll(p.Data)
	if err != nil {
		return fmt.Errorf("error reading file data: %s", err.Error())
	}

	mimetype := http.DetectContentType(data)
	if mimetype != "image/jpeg" && mimetype != "image/png" {
		return fmt.Errorf("invalid file format. only .jpg & .png images allowed")
	}

	if len(data) > 2000000 {
		return fmt.Errorf("file size too large. max size is 2Mb")
	}

	path, err := r.repo.Store().Put(memfs.NewMemfileBytes(p.Filename, data), true)
	if err != nil {
		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	pro := &Profile{
		Poster: path.String(),
	}

	if err := r.saveProfile(pro, res); err != nil {
		return fmt.Errorf("error saving profile: %s", err)
	}

	return nil
}

// saveProfile stores changes to this peer's profile
// does not check to see if inputs are valid
func (r *ProfileRequests) saveProfile(p *Profile, res *Profile) error {
	if p == nil {
		return fmt.Errorf("profile required for update")
	}
	pro := &Profile{}
	if err := r.GetProfile(nil, pro); err != nil {
		return fmt.Errorf("error getting previous profile: %s", err)
	}

	pro.AssignEditable(p)

	// validate new Profile inputs
	if err := p.ValidateProfile(); err != nil {
		return fmt.Errorf("error validating profile: %s", err)
	}

	_p, err := unmarshalProfile(pro)
	if err != nil {
		return err
	}

	if err := r.repo.SaveProfile(_p); err != nil {
		return err
	}

	profile, err := r.repo.Profile()
	if err != nil {
		return err
	}
	p2, err := marshalProfile(profile)
	if err != nil {
		return err
	}

	*res = *p2
	return nil

}
