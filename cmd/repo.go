package cmd

import (
	"net"
	"net/rpc"
	"strings"

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

// RepoOrClient returns either a
func RepoOrClient(online bool) (repo.Repo, *rpc.Client) {
	if fs, err := ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = viper.GetString(IpfsFsPath)
		cfg.Online = online
	}); err == nil {
		id := ""
		if fs.Node().PeerHost != nil {
			id = fs.Node().PeerHost.ID().Pretty()
		}

		r, err := fs_repo.NewRepo(fs, viper.GetString(QriRepoPath), id)
		ExitIfErr(err)
		return r, nil
	} else if strings.Contains(err.Error(), "lock") {
		// TODO - bad bad hardcode
		conn, err := net.Dial("tcp", ":2504")
		if err != nil {
			ErrExit(err)
		}
		return nil, rpc.NewClient(conn)
	} else {
		ErrExit(err)
	}
	return nil, nil
}

func GetIpfsFilestore(online bool) *ipfs.Filestore {
	fs, err := ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = viper.GetString(IpfsFsPath)
		cfg.Online = online
	})
	ExitIfErr(err)
	return fs
}
