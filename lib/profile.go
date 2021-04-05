package lib

import (
	"context"
	"fmt"
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
	d dispatcher
}

// Name returns the name of this method group
func (m ProfileMethods) Name() string {
	return "profile"
}

// Attributes defines attributes for each method
func (m ProfileMethods) Attributes() map[string]AttributeSet {
	return map[string]AttributeSet{
		"getprofile":      {denyRPC, ""},
		"setprofile":      {denyRPC, ""},
		"setprofilephoto": {denyRPC, ""},
		"setposterphoto":  {denyRPC, ""},
	}
}

// ProfileParams define parameters for getting a profile
type ProfileParams struct{}

// GetProfile get's this node's peer profile
func (m ProfileMethods) GetProfile(ctx context.Context, p *ProfileParams) (*config.ProfilePod, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "getprofile"), p)
	if res, ok := got.(*config.ProfilePod); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// SetProfileParams defines parameters for setting parts of a profile
// Cannot use this to set private keys, your peername, or peer id
type SetProfileParams struct {
	Pro *config.ProfilePod
}

// SetProfile stores changes to the active peer's editable profile
func (m ProfileMethods) SetProfile(ctx context.Context, p *SetProfileParams) (*config.ProfilePod, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "setprofile"), p)
	if res, ok := got.(*config.ProfilePod); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// FileParams defines parameters for Files as arguments to lib methods
// either `Filename` or `Data` is required. If both fields are set, the content in the `Data` field is favored
type FileParams struct {
	// url to download data from. either Url or Data is required
	// Url      string
	// Filename of data file. extension is used for filetype detection
	Filename string `qri:"fspath"`
	// Data is the file as slice of bytes
	Data []byte
}

// SetProfilePhoto changes the active peer's profile image
func (m ProfileMethods) SetProfilePhoto(ctx context.Context, p *FileParams) (*config.ProfilePod, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "setprofilephoto"), p)
	if res, ok := got.(*config.ProfilePod); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// SetPosterPhoto changes this active peer's poster image
func (m ProfileMethods) SetPosterPhoto(ctx context.Context, p *FileParams) (*config.ProfilePod, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "setposterphoto"), p)
	if res, ok := got.(*config.ProfilePod); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// profileImpl holds the method implementations for ProfileMethods
type profileImpl struct{}

// GetProfile get's this node's peer profile
func (profileImpl) GetProfile(scope scope, p *ProfileParams) (*config.ProfilePod, error) {
	pro := scope.ActiveProfile()
	cfg := scope.Config()
	// TODO (b5) - this isn't the right way to check if you're online
	if cfg != nil && cfg.P2P != nil {
		pro.Online = cfg.P2P.Enabled
	}

	enc, err := pro.Encode()
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	enc.PrivKey = ""
	return enc, nil
}

// SetProfile stores changes to the active peer's editable profile
func (profileImpl) SetProfile(scope scope, p *SetProfileParams) (*config.ProfilePod, error) {
	if p.Pro == nil {
		return nil, fmt.Errorf("profile required for update")
	}

	pro := p.Pro
	cfg := scope.Config()
	r := scope.Repo()

	cfg.Set("profile.name", pro.Name)
	cfg.Set("profile.email", pro.Email)
	cfg.Set("profile.description", pro.Description)
	cfg.Set("profile.homeurl", pro.HomeURL)
	cfg.Set("profile.twitter", pro.Twitter)

	if pro.Color != "" {
		cfg.Set("profile.color", pro.Color)
	}
	// TODO (b5) - strange bug:
	if cfg.Profile.Type == "" {
		cfg.Profile.Type = "peer"
	}

	prevPeername := cfg.Profile.Peername
	if pro.Peername != "" && pro.Peername != cfg.Profile.Peername {
		cfg.Set("profile.peername", pro.Peername)
	}

	if err := cfg.Profile.Validate(); err != nil {
		return nil, err
	}

	if pro.Peername != "" && pro.Peername != prevPeername {
		if reg := scope.RegistryClient(); reg != nil {
			current, err := profile.NewProfile(cfg.Profile)
			if err != nil {
				return nil, err
			}

			if _, err := reg.PutProfile(&registry.Profile{Username: pro.Peername}, current.PrivKey); err != nil {
				return nil, err
			}
		}
	}

	enc, err := profile.NewProfile(cfg.Profile)
	if err != nil {
		return nil, err
	}
	if err := r.Profiles().SetOwner(enc); err != nil {
		return nil, err
	}

	res := &config.ProfilePod{}
	// Copy the global config, except without the private key.
	*res = *cfg.Profile
	res.PrivKey = ""

	// TODO (b5) - we should have a betteer way of determining onlineness
	if cfg.P2P != nil {
		res.Online = cfg.P2P.Enabled
	}

	if err := scope.ChangeConfig(cfg); err != nil {
		return nil, err
	}
	return res, nil
}

// SetProfilePhoto changes the active peer's profile image
func (profileImpl) SetProfilePhoto(scope scope, p *FileParams) (*config.ProfilePod, error) {
	if err := loadAndValidateJPEG(p, 256000); err != nil {
		return nil, err
	}

	// TODO - if file extension is .jpg / .jpeg ipfs does weird shit that makes this not work
	path, err := scope.Filesystem().DefaultWriteFS().Put(scope.Context(), qfs.NewMemfileBytes("plz_just_encode", p.Data))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error saving photo: %s", err.Error())
	}

	cfg := scope.Config().Copy()
	cfg.Set("profile.photo", path)
	// TODO - resize photo for thumb
	cfg.Set("profile.thumb", path)
	if err := scope.ChangeConfig(cfg); err != nil {
		return nil, err
	}

	pro := scope.ActiveProfile()
	pro.Photo = path
	pro.Thumb = path

	if err := scope.Profiles().SetOwner(pro); err != nil {
		return nil, err
	}

	pp, err := pro.Encode()
	if err != nil {
		return nil, fmt.Errorf("error encoding new profile: %s", err)
	}

	return pp, nil
}

// SetPosterPhoto changes the active peer's poster image
func (profileImpl) SetPosterPhoto(scope scope, p *FileParams) (*config.ProfilePod, error) {
	if err := loadAndValidateJPEG(p, 2<<20); err != nil {
		return nil, err
	}

	// TODO - if file extension is .jpg / .jpeg ipfs does weird shit that makes this not work
	path, err := scope.Filesystem().DefaultWriteFS().Put(scope.Context(), qfs.NewMemfileBytes("plz_just_encode", p.Data))
	if err != nil {
		log.Debug(err.Error())
		return nil, fmt.Errorf("error saving photo: %s", err.Error())
	}

	cfg := scope.Config().Copy()
	cfg.Set("profile.poster", path)
	if err := scope.ChangeConfig(cfg); err != nil {
		return nil, err
	}

	pro := scope.ActiveProfile()
	pro.Poster = path
	if err := scope.Profiles().SetOwner(pro); err != nil {
		return nil, err
	}

	pp, err := pro.Encode()
	if err != nil {
		return nil, fmt.Errorf("error encoding new profile: %s", err)
	}

	return pp, nil
}

func loadAndValidateJPEG(p *FileParams, maxBytes int) (err error) {
	if p.Filename == "" && (p.Data == nil || len(p.Data) == 0) {
		return fmt.Errorf("filename or data required")
	}
	if p.Data == nil || len(p.Data) == 0 {
		if p.Data, err = ioutil.ReadFile(p.Filename); err != nil {
			return fmt.Errorf("error opening file: %w", err)
		}
	}

	if len(p.Data) > maxBytes {
		return fmt.Errorf("file size too large. max size is %s", byteCount(int64(maxBytes)))

	} else if len(p.Data) == 0 {
		return fmt.Errorf("file is empty")
	}

	mimetype := http.DetectContentType(p.Data)
	if mimetype != "image/jpeg" {
		return fmt.Errorf("invalid file format. only .jpg images allowed")
	}
	return nil
}

// provides human readable byte count
func byteCount(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}
