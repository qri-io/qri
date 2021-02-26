package lib

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/registry"
)

// ProfileMethods encapsulates business logic for this node's
// user profile
// TODO (b5) - alterations to user profile are a subset of configuration
// changes. all of this code should be refactored into subroutines of general
// configuration getters & setters
type ProfileMethods struct {
	inst *Instance
}

// CoreRequestsName implements the Request interface
func (ProfileMethods) CoreRequestsName() string { return "profile" }

// NewProfileMethods creates a ProfileMethods pointer from either a repo
// or an rpc.Client
func NewProfileMethods(inst *Instance) *ProfileMethods {
	return &ProfileMethods{inst: inst}
}

// GetProfile get's this node's peer profile
func (m *ProfileMethods) GetProfile(ctx context.Context, in *bool, res *config.ProfilePod) (err error) {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("ProfileMethods.GetProfile", in, res))
	}

	// r := m.inst.repo
	// // TODO - this is a carry-over from when GetProfile only supported getting
	// if res.ID == "" && res.Peername == "" {
	// 	pro = r.Profiles().Owner()
	// } else {
	// 	pro, err = getProfile(ctx, r.Profiles(), res.ID, res.Peername)
	// }

	pro, err := profile.FromCtx(ctx, m.inst.profiles)
	if err != nil {
		return err
	}

	cfg := m.inst.cfg
	// TODO (b5) - this isn't the right way to check if you're online
	if cfg != nil && cfg.P2P != nil {
		pro.Online = cfg.P2P.Enabled
	}

	enc, err := pro.Encode()
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	*res = *enc
	return nil
}

func getProfile(ctx context.Context, pros profile.Store, idStr, peername string) (pro *profile.Profile, err error) {
	if idStr == "" {
		// TODO(b5): we're handling the "me" keyword here, should be handled as part of
		// request scope construction
		if peername == "me" {
			return pros.Owner(), nil
		}
		return profile.ResolveUsername(pros, peername)
	}

	id, err := profile.IDB58Decode(idStr)
	if err != nil {
		log.Debugw("decoding profile ID", "err", err)
		return nil, err
	}
	return pros.GetProfile(id)
}

// SaveProfile stores changes to this peer's editable profile
func (m *ProfileMethods) SaveProfile(p *config.ProfilePod, res *config.ProfilePod) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("ProfileMethods.SaveProfile", p, res))
	}
	if p == nil {
		return fmt.Errorf("profile required for update")
	}

	cfg := m.inst.cfg
	r := m.inst.repo

	if p.Peername != cfg.Profile.Peername && p.Peername != "" {
		if reg := m.inst.registry; reg != nil {
			current, err := profile.NewProfile(cfg.Profile)
			if err != nil {
				return err
			}

			if _, err := reg.PutProfile(&registry.Profile{Username: p.Peername}, current.PrivKey); err != nil {
				return err
			}
		}

		cfg.Set("profile.peername", p.Peername)
	}

	cfg.Set("profile.name", p.Name)
	cfg.Set("profile.email", p.Email)
	cfg.Set("profile.description", p.Description)
	cfg.Set("profile.homeurl", p.HomeURL)
	cfg.Set("profile.twitter", p.Twitter)

	if p.Color != "" {
		cfg.Set("profile.color", p.Color)
	}
	// TODO (b5) - strange bug:
	if cfg.Profile.Type == "" {
		cfg.Profile.Type = "peer"
	}

	pro, err := profile.NewProfile(cfg.Profile)
	if err != nil {
		return err
	}
	if err := r.Profiles().SetOwner(pro); err != nil {
		return err
	}

	// Copy the global config, except without the private key.
	*res = *cfg.Profile
	res.PrivKey = ""

	// TODO (b5) - we should have a betteer way of determining onlineness
	if cfg.P2P != nil {
		res.Online = cfg.P2P.Enabled
	}

	return m.inst.ChangeConfig(cfg)
}

// ProfilePhoto fetches the byte slice of a given user's profile photo
func (m *ProfileMethods) ProfilePhoto(req *config.ProfilePod, res *[]byte) (err error) {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("ProfileMethods.ProfilePhoto", req, res))
	}
	ctx := context.TODO()

	r := m.inst.repo

	pro, e := getProfile(ctx, r.Profiles(), req.ID, req.Peername)
	if e != nil {
		return e
	}

	if pro.Photo == "" || pro.Photo == "/" {
		return nil
	}

	f, e := r.Filesystem().Get(ctx, pro.Photo)
	if e != nil {
		return e
	}

	*res, err = ioutil.ReadAll(f)
	return
}

// FileParams defines parameters for Files as arguments to lib methods
type FileParams struct {
	// Url      string    // url to download data from. either Url or Data is required
	Filename string    // filename of data file. extension is used for filetype detection
	Data     io.Reader // reader of structured data. either Url or Data is required
}

// SetProfilePhoto changes this peer's profile image
func (m *ProfileMethods) SetProfilePhoto(p *FileParams, res *config.ProfilePod) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("ProfileMethods.SetProfilePhoto", p, res))
	}
	ctx := context.TODO()

	r := m.inst.repo

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
	path, err := r.Filesystem().DefaultWriteFS().Put(ctx, qfs.NewMemfileBytes("plz_just_encode", data))
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	res.Photo = path
	res.Thumb = path
	cfg := m.inst.cfg.Copy()
	cfg.Set("profile.photo", path)
	// TODO - resize photo for thumb
	cfg.Set("profile.thumb", path)
	if err := m.inst.ChangeConfig(cfg); err != nil {
		return err
	}

	pro := r.Profiles().Owner()
	pro.Photo = path
	pro.Thumb = path

	if err := r.Profiles().SetOwner(pro); err != nil {
		return err
	}

	pp, err := pro.Encode()
	if err != nil {
		return fmt.Errorf("error encoding new profile: %s", err)
	}

	*res = *pp
	return nil
}

// PosterPhoto fetches the byte slice of a given user's poster photo
func (m *ProfileMethods) PosterPhoto(req *config.ProfilePod, res *[]byte) (err error) {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("ProfileMethods.PostPhoto", req, res))
	}
	ctx := context.TODO()

	r := m.inst.repo
	pro, e := getProfile(ctx, r.Profiles(), req.ID, req.Peername)
	if e != nil {
		return e
	}

	if pro.Poster == "" || pro.Poster == "/" {
		return nil
	}

	f, e := r.Filesystem().Get(ctx, pro.Poster)
	if e != nil {
		return e
	}

	*res, err = ioutil.ReadAll(f)
	return
}

// SetPosterPhoto changes this peer's poster image
func (m *ProfileMethods) SetPosterPhoto(p *FileParams, res *config.ProfilePod) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("ProfileMethods.SetPosterPhoto", p, res))
	}
	ctx := context.TODO()

	if p.Data == nil {
		return fmt.Errorf("file is required")
	}

	r := m.inst.repo

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
	path, err := r.Filesystem().DefaultWriteFS().Put(ctx, qfs.NewMemfileBytes("plz_just_encode", data))
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	res.Poster = path
	cfg := m.inst.cfg.Copy()
	cfg.Set("profile.poster", path)
	if err := m.inst.ChangeConfig(cfg); err != nil {
		return err
	}

	pro := r.Profiles().Owner()
	pro.Poster = path
	if err := r.Profiles().SetOwner(pro); err != nil {
		return err
	}

	pp, err := pro.Encode()
	if err != nil {
		return fmt.Errorf("error encoding new profile: %s", err)
	}

	*res = *pp
	return nil
}
