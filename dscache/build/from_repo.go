package build

import (
	"context"

	"github.com/qri-io/qri/dscache"
	"github.com/qri-io/qri/repo"
)

// DscacheFromRepo creates a dscache, building it from the repo's logbook and profiles and dsrefs.
func DscacheFromRepo(ctx context.Context, r repo.Repo) (*dscache.Dscache, error) {
	num, err := r.RefCount()
	if err != nil {
		return nil, err
	}
	refs, err := r.References(0, num)
	if err != nil {
		return nil, err
	}
	profiles := r.Profiles()
	logbook := r.Logbook()
	store := r.Store()
	filesys := r.Filesystem()
	return dscache.BuildDscacheFromLogbookAndProfilesAndDsref(ctx, refs, profiles, logbook, store, filesys)
}
