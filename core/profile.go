package core

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/rpc"
	"time"

	"github.com/qri-io/cafs/memfs"
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

// SaveProfile stores changes to this peer's profile
func (r *ProfileRequests) SaveProfile(p *Profile, res *Profile) error {
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.SaveProfile", p, res)
	}
	if p == nil {
		return fmt.Errorf("profile required for update")
	}

	_p, err := unmarshalProfile(p)
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

	pro, err := r.repo.Profile()
	if err != nil {
		return fmt.Errorf("error loading profile: %s", err.Error())
	}

	path, err := r.repo.Store().Put(memfs.NewMemfileBytes(p.Filename, data), true)
	if err != nil {
		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	pro.Profile = path
	// TODO - resize photo
	pro.Thumb = path

	if err := r.repo.SaveProfile(pro); err != nil {
		return fmt.Errorf("error saving profile: %s", err.Error())
	}

	_p, err := marshalProfile(pro)
	if err != nil {
		return err
	}

	*res = *_p
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

	pro, err := r.repo.Profile()
	if err != nil {
		return fmt.Errorf("error loading profile: %s", err.Error())
	}

	path, err := r.repo.Store().Put(memfs.NewMemfileBytes(p.Filename, data), true)
	if err != nil {
		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	pro.Poster = path

	if err := r.repo.SaveProfile(pro); err != nil {
		return fmt.Errorf("error saving profile: %s", err.Error())
	}

	_p, err := marshalProfile(pro)
	if err != nil {
		return err
	}

	*res = *_p
	return nil
}
