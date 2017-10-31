package core

import (
	"fmt"

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
