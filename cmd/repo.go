package cmd

import (
	"fmt"
	"net"
	"net/rpc"
	"strings"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/core"
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

func DatasetRequests(online bool) (*core.DatasetRequests, error) {
	r, cli, err := RepoOrClient(online)
	if err != nil {
		return nil, err
	}
	return core.NewDatasetRequests(r, cli), nil
}

// RepoOrClient returns either a
func RepoOrClient(online bool) (repo.Repo, *rpc.Client, error) {
	if fs, err := ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = viper.GetString(IpfsFsPath)
		cfg.Online = online
	}); err == nil {
		id := ""
		if fs.Node().PeerHost != nil {
			id = fs.Node().PeerHost.ID().Pretty()
		}

		r, err := fs_repo.NewRepo(fs, viper.GetString(QriRepoPath), id)
		return r, nil, err

	} else if strings.Contains(err.Error(), "lock") {
		// TODO - bad bad hardcode
		conn, err := net.Dial("tcp", ":2504")
		if err != nil {
			return nil, nil, err
		}
		return nil, rpc.NewClient(conn), nil
	} else {
		return nil, nil, err
	}

	return nil, nil, fmt.Errorf("badbadnotgood")
}
