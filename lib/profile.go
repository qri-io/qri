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

// ProfileRequests encapsulates business logic for this node's
// user profile
type ProfileRequests struct {
	node *p2p.QriNode
	cli  *rpc.Client
}

// CoreRequestsName implements the Request interface
func (ProfileRequests) CoreRequestsName() string { return "profile" }

// NewProfileRequests creates a ProfileRequests pointer from either a repo
// or an rpc.Client
func NewProfileRequests(node *p2p.QriNode, cli *rpc.Client) *ProfileRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewProfileRequests"))
	}

	return &ProfileRequests{
		node: node,
		cli:  cli,
	}
}

// GetProfile get's this node's peer profile
func (r *ProfileRequests) GetProfile(in *bool, res *config.ProfilePod) (err error) {
	var pro *profile.Profile
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.GetProfile", in, res)
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

	if Config != nil && Config.P2P != nil {
		pro.Online = Config.P2P.Enabled
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
		if err = repo.CanonicalizeProfile(r.node.Repo, ref, nil); err != nil {
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
func (r *ProfileRequests) SaveProfile(p *config.ProfilePod, res *config.ProfilePod) error {
	if r.cli != nil {
		return r.cli.Call("ProfileRequests.SaveProfile", p, res)
	}
	if p == nil {
		return fmt.Errorf("profile required for update")
	}

	if p.Peername != Config.Profile.Peername && p.Peername != "" {
		// TODO - should ProfileRequests be allocated with a configuration? How should this work in relation to
		// RPC requests?
		if reg := r.node.Repo.Registry(); reg != nil {
			current, err := profile.NewProfile(Config.Profile)
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

		Config.Set("profile.peername", p.Peername)
	}

	Config.Set("profile.name", p.Name)
	Config.Set("profile.email", p.Email)
	Config.Set("profile.description", p.Description)
	Config.Set("profile.homeurl", p.HomeURL)
	Config.Set("profile.twitter", p.Twitter)

	if p.Color != "" {
		Config.Set("profile.color", p.Color)
	}
	// TODO - strange bug:
	if Config.Profile.Type == "" {
		Config.Profile.Type = "peer"
	}

	pro, err := profile.NewProfile(Config.Profile)
	if err != nil {
		return err
	}
	if err := r.node.Repo.SetProfile(pro); err != nil {
		return err
	}

	// Copy the global config, except without the private key.
	*res = *Config.Profile
	res.PrivKey = ""

	if Config.P2P != nil {
		res.Online = Config.P2P.Enabled
	}

	return SetConfig(Config)
}

// ProfilePhoto fetches the byte slice of a given user's profile photo
func (r *ProfileRequests) ProfilePhoto(req *config.ProfilePod, res *[]byte) (err error) {
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
func (r *ProfileRequests) SetProfilePhoto(p *FileParams, res *config.ProfilePod) error {
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
	path, err := r.node.Repo.Store().Put(qfs.NewMemfileBytes("plz_just_encode", data), true)
	if err != nil {
		log.Debug(err.Error())
		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	res.Photo = path
	res.Thumb = path
	Config.Set("profile.photo", path)
	// TODO - resize photo for thumb
	Config.Set("profile.thumb", path)

	pro, err := profile.NewProfile(Config.Profile)
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

	return SetConfig(Config)
}

// PosterPhoto fetches the byte slice of a given user's poster photo
func (r *ProfileRequests) PosterPhoto(req *config.ProfilePod, res *[]byte) (err error) {
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
func (r *ProfileRequests) SetPosterPhoto(p *FileParams, res *config.ProfilePod) error {
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
	path, err := r.node.Repo.Store().Put(qfs.NewMemfileBytes("plz_just_encode", data), true)
	if err != nil {
		log.Debug(err.Error())

		return fmt.Errorf("error saving photo: %s", err.Error())
	}

	res.Poster = path
	Config.Set("profile.poster", path)

	pro, err := profile.NewProfile(Config.Profile)
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

	return SetConfig(Config)
}
