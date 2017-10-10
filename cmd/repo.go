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

func GetRepo(online bool) repo.Repo {
	if r != nil {
		return r
	}

	fs, err := GetIpfsFilestore(online)
	ExitIfErr(err)

	r, err := fs_repo.NewRepo(fs, viper.GetString(QriRepoPath))
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

func DatasetRef(r repo.Repo, store cafs.Filestore, arg string) (*repo.DatasetRef, error) {
	path, err := r.GetPath(arg)
	if err != nil {
		return nil, err
	}

	ds, err := dsfs.LoadDataset(store, path)
	if err != nil {
		return nil, err
	}
	// TODO - add hash lookup

	name, err := r.GetName(path)
	if err != nil {
		return nil, err
	}

	return &repo.DatasetRef{
		Path:    path,
		Name:    name,
		Dataset: ds,
	}, nil
}
