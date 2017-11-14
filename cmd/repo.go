package cmd

import (
	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
	"github.com/spf13/viper"
)

var r repo.Repo

func GetRepo(online bool) repo.Repo {
	if r != nil {
		return r
	}

	fs := GetIpfsFilestore(online)
	id := ""
	if fs.Node().PeerHost != nil {
		id = fs.Node().PeerHost.ID().Pretty()
	}

	r, err := fs_repo.NewRepo(fs, viper.GetString(QriRepoPath), id)
	ExitIfErr(err)
	return r
}

func GetIpfsFilestore(online bool) *ipfs.Filestore {
	fs, err := ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = viper.GetString(IpfsFsPath)
		cfg.Online = online
	})
	ExitIfErr(err)
	return fs
}
