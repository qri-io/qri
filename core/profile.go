package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/rpc"
	"strings"

	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regclient"
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

// GetProfile get's this node's peer profile
func (r *ProfileRequests) GetProfile(in *bool, res *profile.CodingProfile) (err error) {
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

	enc, err := pro.Encode()
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	*res = *enc
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

// SaveProfile stores changes to this peer's editable profile profile
func (r *ProfileRequests) SaveProfile(p *profile.CodingProfile, res *profile.CodingProfile) error {
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.SaveProfile", p, res)
	}
	if p == nil {
		return fmt.Errorf("profile required for update")
	}

	if p.Peername != "" {
		// TODO - should ProfileRequests be allocated with a configuration? How should this work in relation to
		// RPC requests?
		if Config.Registry != nil {
			current, err := profile.NewProfile(Config.Profile)
			if err != nil {
				return err
			}

			reg := regclient.NewClient(&regclient.Config{
				Location: Config.Registry.Location,
			})

			if err := reg.PutProfile(p.Peername, current.PrivKey); err != nil {
				if strings.Contains(err.Error(), "taken") {
					return ErrHandleTaken
				}
				return err
			}
		}

		Config.Set("profile.peername", p.Peername)
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

	// TODO - strange bug:
	if Config.Profile.Type == "" {
		Config.Profile.Type = "peer"
	}

	pro, err := profile.NewProfile(Config.Profile)
	if err != nil {
		return err
	}
	if err := r.repo.Profiles().PutProfile(pro); err != nil {
		return err
	}

	_res := profile.NewCodingProfile(Config.Profile)
	_res.PrivKey = ""
	*res = *_res

	return SaveConfig()
}

// ProfilePhoto fetches the byte slice of a given user's profile photo
func (r *ProfileRequests) ProfilePhoto(req *profile.CodingProfile, res *[]byte) (err error) {
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
func (r *ProfileRequests) SetProfilePhoto(p *FileParams, res *profile.CodingProfile) error {
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
func (r *ProfileRequests) PosterPhoto(req *profile.CodingProfile, res *[]byte) (err error) {
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
func (r *ProfileRequests) SetPosterPhoto(p *FileParams, res *profile.CodingProfile) error {
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
