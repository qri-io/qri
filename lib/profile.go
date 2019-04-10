package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/rpc"
	"strings"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// ProfileMethods access and edit a Qri profile
type ProfileMethods interface {
	Methods
	ProfileReads
	ProfileWrites
}

type ProfileReads interface {
	GetProfile(in *bool, res *config.ProfilePod) error
	PosterPhoto(req *config.ProfilePod, res *[]byte) error
	ProfilePhoto(req *config.ProfilePod, res *[]byte) error
}

type ProfileWrites interface {
	SaveProfile(p *config.ProfilePod, res *config.ProfilePod) error
	SetProfilePhoto(p *FileParams, res *config.ProfilePod) error
	SetPosterPhoto(p *FileParams, res *config.ProfilePod) error
}

// NewProfileMethods creates a profileMethods pointer from either a repo
// or an rpc.Client
func NewProfileMethods(inst Instance) ProfileMethods {
	return profileMethods{
		inst: inst,
		node: inst.Node(),
		cfg:  inst.Config(),
		cli:  inst.RPC(),
	}
}

// profileMethods encapsulates business logic for this node's
// user profile
type profileMethods struct {
	inst        Instance
	node        *p2p.QriNode
	cfg         *config.Config
	cfgFilepath string

	cli *rpc.Client
}

// MethodsKind implements the Request interface
func (profileMethods) MethodsKind() string { return "ProfileMethods" }

// GetProfile get's this node's peer profile
func (r profileMethods) GetProfile(in *bool, res *config.ProfilePod) (err error) {
	var pro *profile.Profile
	if r.cli != nil {
		return r.cli.Call("ProfileMethods.GetProfile", in, res)
	}

	// TODO - this is a carry-over from when GetProfile only supported getting
	if res.ID == "" && res.Peername == "" {
		pro, err = r.node.Repo.Profile()
	} else {
		pro, err = r.getProfile(res.ID, res.Peername)
	}

	if err != nil {
		return err
	}

	if r.cfg != nil && r.cfg.P2P != nil {
		pro.Online = r.cfg.P2P.Enabled
	}

	enc, err := pro.Encode()
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	*res = *enc
	return nil
}

func (r profileMethods) getProfile(idStr, peername string) (pro *profile.Profile, err error) {
	var id profile.ID
	if idStr == "" {
		ref := &repo.DatasetRef{
			Peername: peername,
		}
		if err = repo.CanonicalizeProfile(r.node.Repo, ref); err != nil {
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
	profile, err := r.node.Repo.Profile()
	if err == nil && profile.ID.String() == id.String() {
		return profile, nil
	}

	return r.node.Repo.Profiles().GetProfile(id)
}

// SaveProfile stores changes to this peer's editable profile
func (r profileMethods) SaveProfile(p *config.ProfilePod, res *config.ProfilePod) error {
	if r.cli != nil {
		return r.cli.Call("profileMethods.SaveProfile", p, res)
	}
	if p == nil {
		return fmt.Errorf("profile required for update")
	}
	cfg := r.cfg.Copy()

	if p.Peername != cfg.Profile.Peername && p.Peername != "" {
		// TODO - should profileMethods be allocated with a configuration? How should this work in relation to
		// RPC requests?
		if reg := r.node.Repo.Registry(); reg != nil {
			current, err := profile.NewProfile(cfg.Profile)
			if err != nil {
				return err
			}

			if err := reg.PutProfile(p.Peername, current.PrivKey); err != nil {
				if strings.Contains(err.Error(), "taken") {
					return actions.ErrHandleTaken
				}
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
	// TODO - strange bug:
	if cfg.Profile.Type == "" {
		cfg.Profile.Type = "peer"
	}

	pro, err := profile.NewProfile(cfg.Profile)
	if err != nil {
		return err
	}
	if err := r.node.Repo.SetProfile(pro); err != nil {
		return err
	}

	// Copy the global config, except without the private key.
	*res = *cfg.Profile
	res.PrivKey = ""

	if cfg.P2P != nil {
		res.Online = cfg.P2P.Enabled
	}

	writable, ok := r.inst.(WritableInstance)
	if !ok {
		return ErrNotWritable
	}
	return writable.SetConfig(cfg)
}

// ProfilePhoto fetches the byte slice of a given user's profile photo
func (r profileMethods) ProfilePhoto(req *config.ProfilePod, res *[]byte) (err error) {
	pro, e := r.getProfile(req.ID, req.Peername)
	if e != nil {
		return e
	}

	if pro.Photo == "" || pro.Photo == "/" {
		return nil
	}

	f, e := r.node.Repo.Store().Get(pro.Photo)
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
func (r profileMethods) SetProfilePhoto(p *FileParams, res *config.ProfilePod) error {
	if r.cli != nil {
		return r.cli.Call("profileMethods.SetProfilePhoto", p, res)
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
	path, err := r.node.Repo.Store().Put(qfs.NewMemfileBytes("plz_just_encode", data), true)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	res.Photo = path
	res.Thumb = path
	cfg := r.cfg.Copy()
	cfg.Set("profile.photo", path)
	// TODO - resize photo for thumb
	cfg.Set("profile.thumb", path)

	pro, err := profile.NewProfile(cfg.Profile)
	if err != nil {
		return err
	}
	if err := r.node.Repo.SetProfile(pro); err != nil {
		return err
	}

	newPro, err := r.node.Repo.Profile()
	if err != nil {
		return fmt.Errorf("error getting newly set profile: %s", err)
	}
	pp, err := newPro.Encode()
	if err != nil {
		return fmt.Errorf("error encoding new profile: %s", err)
	}

	*res = *pp

	writable, ok := r.inst.(WritableInstance)
	if !ok {
		return ErrNotWritable
	}
	return writable.SetConfig(cfg)
}

// PosterPhoto fetches the byte slice of a given user's poster photo
func (r profileMethods) PosterPhoto(req *config.ProfilePod, res *[]byte) (err error) {
	pro, e := r.getProfile(req.ID, req.Peername)
	if e != nil {
		return e
	}

	if pro.Poster == "" || pro.Poster == "/" {
		return nil
	}

	f, e := r.node.Repo.Store().Get(pro.Poster)
	if e != nil {
		return e
	}

	*res, err = ioutil.ReadAll(f)
	return
}

// SetPosterPhoto changes this peer's poster image
func (r profileMethods) SetPosterPhoto(p *FileParams, res *config.ProfilePod) error {
	if r.cli != nil {
		return r.cli.Call("profileMethods.SetPosterPhoto", p, res)
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
	path, err := r.node.Repo.Store().Put(qfs.NewMemfileBytes("plz_just_encode", data), true)
	if err != nil {
		log.Debug(err.Error())

		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	res.Poster = path
	cfg := r.cfg.Copy()
	cfg.Set("profile.poster", path)

	pro, err := profile.NewProfile(cfg.Profile)
	if err != nil {
		return err
	}
	if err := r.node.Repo.SetProfile(pro); err != nil {
		return err
	}

	newPro, err := r.node.Repo.Profile()
	if err != nil {
		return fmt.Errorf("error getting newly set profile: %s", err)
	}
	pp, err := newPro.Encode()
	if err != nil {
		return fmt.Errorf("error encoding new profile: %s", err)
	}

	*res = *pp

	writable, ok := r.inst.(WritableInstance)
	if !ok {
		return ErrNotWritable
	}
	return writable.SetConfig(cfg)
}
