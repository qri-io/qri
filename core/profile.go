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

	"github.com/qri-io/cafs"
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
	ID          string       `json:"id"`
	Created     time.Time    `json:"created,omitempty"`
	Updated     time.Time    `json:"updated,omitempty"`
	Peername    string       `json:"peername"`
	Type        profile.Type `json:"type"`
	Email       string       `json:"email"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	HomeURL     string       `json:"homeUrl"`
	Color       string       `json:"color"`
	Thumb       string       `json:"thumb"`
	Photo       string       `json:"photo"`
	Poster      string       `json:"poster"`
	Twitter     string       `json:"twitter"`
}

// unmarshalProfile gets a profile.Profile from a Profile pointer
func unmarshalProfile(p *Profile) (*profile.Profile, error) {
	// silly workaround for gob encoding
	data, err := json.Marshal(p)
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("err re-encoding json: %s", err.Error())
	}

	_p := &profile.Profile{}
	if err := json.Unmarshal(data, _p); err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error unmarshaling json: %s", err.Error())
	}

	return _p, nil
}

// marshalProfile gets a Profile pointer from a profile.Profile pointer
func marshalProfile(p *profile.Profile) (*Profile, error) {
	// silly workaround for gob encoding
	data, err := json.Marshal(p)
	if err != nil {
		log.Error(err.Error())
		return nil, fmt.Errorf("err re-encoding json: %s", err.Error())
	}

	_p := &Profile{}
	if err := json.Unmarshal(data, _p); err != nil {
		log.Error(err.Error())
		return nil, fmt.Errorf("error unmarshaling json: %s", err.Error())
	}

	return _p, nil
}

// GetProfile get's this node's peer profile
func (r *ProfileRequests) GetProfile(in *bool, res *Profile) (err error) {
	var pro *profile.Profile
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.GetProfile", in, res)
	}

	// TODO - this is a carry-over from when GetProfile only supported getting
	if res.ID == "" && res.Peername == "" {
		pro, err = r.repo.Profile()
	} else {
		pro, err = r.getProfile(res.ID, res.Peername)
	}

	if err != nil {
		return err
	}

	_p, err := marshalProfile(pro)
	if err != nil {
		log.Debug(err.Error())
		return err
	}
	*res = *_p
	return nil
}

func (r *ProfileRequests) getProfile(idStr, peername string) (pro *profile.Profile, err error) {
	var id profile.ID
	if idStr == "" {
		ref := &repo.DatasetRef{
			Peername: peername,
		}
		if err = repo.CanonicalizeProfile(r.repo, ref); err != nil {
			log.Error("error canonicalizing profile", err.Error())
			return nil, err
		}
		id = ref.ProfileID
	} else {
		id, err = profile.IDB58Decode(idStr)
		if err != nil {
			log.Error("err decoding multihash", err.Error())
			return nil, err
		}
	}

	// TODO - own profile should just be inside the profile store
	profile, err := r.repo.Profile()
	if err == nil && profile.ID.String() == id.String() {
		return profile, nil
	}

	return r.repo.Profiles().GetProfile(id)
}

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
		if pf.Photo != "" {
			p.Photo = pf.Photo
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
		log.Debug(err.Error())
		return fmt.Errorf("error opening schemas/profileSchema.json: %s", err)
	}
	defer f.Close()

	profileSchema, err := ioutil.ReadAll(f)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error reading profileSchema: %s", err)
	}

	// unmarshal to rootSchema
	rs := &jsonschema.RootSchema{}
	if err := json.Unmarshal(profileSchema, rs); err != nil {
		return fmt.Errorf("error unmarshaling profileSchema to RootSchema: %s", err)
	}
	profile, err := json.Marshal(p)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error marshaling profile to json: %s", err)
	}
	if errors, err := rs.ValidateBytes(profile); len(errors) > 0 {
		return fmt.Errorf("%s", errors)
	} else if err != nil {
		log.Debug(err.Error())
		return err
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

	Config.Set("profile.peername", p.Peername)
	return SaveConfig()
}

// SaveProfile stores changes to this peer's editable profile profile
func (r *ProfileRequests) SaveProfile(p *Profile, res *Profile) error {
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.SaveProfile", p, res)
	}
	if p == nil {
		return fmt.Errorf("profile required for update")
	}

	if p.Name != "" {
		Config.Set("profile.name", p.Name)
	}
	if p.Email != "" {
		Config.Set("profile.email", p.Email)
	}
	if p.Description != "" {
		Config.Set("profile.description", p.Description)
	}
	if p.HomeURL != "" {
		Config.Set("profile.homeurl", p.HomeURL)
	}
	if p.Color != "" {
		Config.Set("profile.color", p.Color)
	}
	if p.Twitter != "" {
		Config.Set("profile.twitter", p.Twitter)
	}

	resp, err := Config.Profile.DecodeProfile()
	if err != nil {
		return err
	}

	*res = Profile{
		ID:          resp.ID.String(),
		Created:     resp.Created,
		Updated:     resp.Updated,
		Peername:    resp.Peername,
		Type:        resp.Type,
		Email:       resp.Email,
		Name:        resp.Name,
		Description: resp.Description,
		HomeURL:     resp.HomeURL,
		Color:       resp.Color,
		Thumb:       resp.Thumb.String(),
		Photo:       resp.Photo.String(),
		Poster:      resp.Poster.String(),
		Twitter:     resp.Twitter,
	}

	return SaveConfig()
}

// ProfilePhoto fetches the byte slice of a given user's profile photo
func (r *ProfileRequests) ProfilePhoto(req *Profile, res *[]byte) (err error) {
	pro, e := r.getProfile(req.ID, req.Peername)
	if e != nil {
		return e
	}

	if pro.Photo.String() == "" || pro.Photo.String() == "/" {
		return nil
	}

	f, e := r.repo.Store().Get(pro.Photo)
	if e != nil {
		return e
	}

	*res, err = ioutil.ReadAll(f)
	return
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

	// TODO - make the reader be a sizefile to avoid this double-read
	data, err := ioutil.ReadAll(p.Data)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error reading file data: %s", err.Error())
	}
	if len(data) > 250000 {
		return fmt.Errorf("file size too large. max size is 250kb")
	} else if len(data) == 0 {
		return fmt.Errorf("data file is empty")
	}

	mimetype := http.DetectContentType(data)
	if mimetype != "image/jpeg" {
		return fmt.Errorf("invalid file format. only .jpg images allowed")
	}

	// TODO - if file extension is .jpg / .jpeg ipfs does weird shit that makes this not work
	path, err := r.repo.Store().Put(cafs.NewMemfileBytes("plz_just_encode", data), true)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	res.Photo = path.String()
	res.Thumb = path.String()
	Config.Set("profile.photo", path.String())
	// TODO - resize photo for thumb
	Config.Set("profile.thumb", path.String())
	return SaveConfig()
}

// PosterPhoto fetches the byte slice of a given user's poster photo
func (r *ProfileRequests) PosterPhoto(req *Profile, res *[]byte) (err error) {
	pro, e := r.getProfile(req.ID, req.Peername)
	if e != nil {
		return e
	}

	if pro.Poster.String() == "" || pro.Poster.String() == "/" {
		return nil
	}

	f, e := r.repo.Store().Get(pro.Poster)
	if e != nil {
		return e
	}

	*res, err = ioutil.ReadAll(f)
	return
}

// SetPosterPhoto changes this peer's poster image
func (r *ProfileRequests) SetPosterPhoto(p *FileParams, res *Profile) error {
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.SetPosterPhoto", p, res)
	}

	if p.Data == nil {
		return fmt.Errorf("file is required")
	}

	// TODO - make the reader be a sizefile to avoid this double-read
	data, err := ioutil.ReadAll(p.Data)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error reading file data: %s", err.Error())
	}

	if len(data) > 2000000 {
		return fmt.Errorf("file size too large. max size is 2Mb")
	} else if len(data) == 0 {
		return fmt.Errorf("file is empty")
	}

	mimetype := http.DetectContentType(data)
	if mimetype != "image/jpeg" {
		return fmt.Errorf("invalid file format. only .jpg images allowed")
	}

	// TODO - if file extension is .jpg / .jpeg ipfs does weird shit that makes this not work
	path, err := r.repo.Store().Put(cafs.NewMemfileBytes("plz_just_encode", data), true)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	res.Poster = path.String()
	Config.Set("profile.poster", path.String())
	return SaveConfig()
}
