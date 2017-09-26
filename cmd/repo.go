package cmd

import (
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
	"github.com/spf13/viper"
)

var r repo.Repo

func GetRepo() repo.Repo {
	if r != nil {
		return r
	}
	r, err := fs_repo.NewRepo(viper.GetString(QriRepoPath))
	ExitIfErr(err)
	return r
}

func FindDataset(r repo.Repo, store cafs.Filestore, arg string) (*dataset.Dataset, error) {
	path, err := r.GetPath(arg)
	if err == nil {
		return dsfs.LoadDataset(store, path)
	}

	// TODO - add lookups by hashes & stuff
	return nil, cafs.ErrNotFound
}
