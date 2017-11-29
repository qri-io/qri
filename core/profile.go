package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

type ProfileRequests struct {
	repo repo.Repo
}

func NewProfileRequests(r repo.Repo) *ProfileRequests {
	return &ProfileRequests{
		repo: r,
	}
}

func (r *ProfileRequests) GetProfile(in *bool, res *profile.Profile) error {
	profile, err := r.repo.Profile()
	if err != nil {
		return err
	}
	*res = *profile
	return nil
}

func (r *ProfileRequests) SaveProfile(p *profile.Profile, res *profile.Profile) error {
	if p == nil {
		return fmt.Errorf("profile required for update")
	}

	if err := r.repo.SaveProfile(p); err != nil {
		return err
	}

	profile, err := r.repo.Profile()
	if err != nil {
		return err
	}

	*res = *profile
	return nil
}

// FileParams
type FileParams struct {
	// Url      string    // url to download data from. either Url or Data is required
	Filename string    // filename of data file. extension is used for filetype detection
	Data     io.Reader // reader of structured data. either Url or Data is required
}

func (r *ProfileRequests) SetProfilePhoto(p *FileParams, res *profile.Profile) error {
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

	*res = *pro
	return nil
}

func (r *ProfileRequests) SetPosterPhoto(p *FileParams, res *profile.Profile) error {
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

	return nil
}
