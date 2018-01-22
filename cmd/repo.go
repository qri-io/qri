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
	// "github.com/libp2p/go-libp2p-crypto"
	// gxcrypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

var r repo.Repo

func getRepo(online bool) repo.Repo {
	if r != nil {
		return r
	}

	if !QRIRepoInitialized() {
		ErrExit(fmt.Errorf("no qri repo found, please run `qri init`"))
	}

	fs := getIpfsFilestore(online)
	id := ""
	if fs.Node().PeerHost != nil {
		id = fs.Node().PeerHost.ID().Pretty()
	}

	r, err := fsrepo.NewRepo(fs, QriRepoPath, id)
	// r.SetPrivateKey(fs.Node().PrivateKey)
	ExitIfErr(err)
	return r
}

func getIpfsFilestore(online bool) *ipfs.Filestore {
	fs, err := ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = IpfsFsPath
		cfg.Online = online
	})
	ExitIfErr(err)
	return fs
}

func datasetRequests(online bool) (*core.DatasetRequests, error) {
	r, cli, err := repoOrClient(online)
	if err != nil {
		return nil, err
	}
	return core.NewDatasetRequests(r, cli), nil
}

func queryRequests(online bool) (*core.QueryRequests, error) {
	r, cli, err := repoOrClient(online)
	if err != nil {
		return nil, err
	}
	return core.NewQueryRequests(r, cli), nil
}

func profileRequests(online bool) (*core.ProfileRequests, error) {
	r, cli, err := repoOrClient(online)
	if err != nil {
		return nil, err
	}
	return core.NewProfileRequests(r, cli), nil
}

func searchRequests(online bool) (*core.SearchRequests, error) {
	r, cli, err := repoOrClient(online)
	if err != nil {
		return nil, err
	}
	return core.NewSearchRequests(r, cli), nil
}

func repoOrClient(online bool) (repo.Repo, *rpc.Client, error) {
	if fs, err := ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = IpfsFsPath
		cfg.Online = online
	}); err == nil {
		// id := ""
		// if fs.Node().PeerHost != nil {
		// 	id = fs.Node().PeerHost.ID().Pretty()
		// }

		cfg, err := readConfigFile()
		ExitIfErr(err)

		r, err := fsrepo.NewRepo(fs, QriRepoPath, cfg.PeerID)
		ExitIfErr(err)

		pk, err := cfg.UnmarshalPrivateKey()
		ExitIfErr(err)

		// if err := fs.Node().LoadPrivateKey(); err != nil {
		// 	return r, nil, err
		// }
		// pk := fs.Node().PrivateKey
		// data, err := pk.Bytes()
		// if err != nil {
		// 	return r, nil, err
		// }
		// pk2, err := crypto.UnmarshalPrivateKey(data)
		// if err != nil {
		// 	return r, nil, err
		// }

		r.SetPrivateKey(pk)
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
