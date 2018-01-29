package cmd

import (
	"fmt"
	"net"
	"net/rpc"
	"strings"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
)

var r repo.Repo

func getRepo(online bool) repo.Repo {
	if r != nil {
		return r
	}

	if !QRIRepoInitialized() {
		ErrExit(fmt.Errorf("no qri repo found, please run `qri setup`"))
	}

	cfg, err := readConfigFile()
	ExitIfErr(err)

	pk, err := cfg.UnmarshalPrivateKey()
	ExitIfErr(err)

	fs := getIpfsFilestore(online)

	r, err := fsrepo.NewRepo(fs, QriRepoPath, cfg.PeerID)
	r.SetPrivateKey(pk)

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

// func queryRequests(online bool) (*core.QueryRequests, error) {
// 	r, cli, err := repoOrClient(online)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return core.NewQueryRequests(r, cli), nil
// }

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

func historyRequests(online bool) (*core.HistoryRequests, error) {
	r, cli, err := repoOrClient(online)
	if err != nil {
		return nil, err
	}
	return core.NewHistoryRequests(r, cli), nil
}

func peerRequests(online bool) (*core.PeerRequests, error) {
	node, err := onlineQriNode()
	if err != nil {
		return nil, err
	}
	return core.NewPeerRequests(node, nil), nil
}

func repoOrClient(online bool) (repo.Repo, *rpc.Client, error) {
	if fs, err := ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = IpfsFsPath
		cfg.Online = online
	}); err == nil {
		cfg, err := readConfigFile()
		ExitIfErr(err)

		r, err := fsrepo.NewRepo(fs, QriRepoPath, cfg.PeerID)
		ExitIfErr(err)

		pk, err := cfg.UnmarshalPrivateKey()
		ExitIfErr(err)

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

func onlineQriNode() (node *p2p.QriNode, err error) {
	var (
		r  repo.Repo
		fs *ipfs.Filestore
	)

	fs, err = ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = IpfsFsPath
		cfg.Online = true
	})

	if err != nil {
		return
	}

	cfg, err := readConfigFile()
	if err != nil {
		return
	}

	r, err = fsrepo.NewRepo(fs, QriRepoPath, cfg.PeerID)
	if err != nil {
		return
	}

	pk, err := cfg.UnmarshalPrivateKey()
	if err != nil {
		return
	}

	r.SetPrivateKey(pk)

	node, err = p2p.NewQriNode(r, func(ncfg *p2p.NodeCfg) {
		ncfg.Logger = log
		ncfg.Online = true
		ncfg.QriBootstrapAddrs = cfg.Bootstrap
	})
	if err != nil {
		return
	}

	log.Info("p2p addresses:")
	for _, a := range node.EncapsulatedAddresses() {
		log.Infof("  %s", a.String())
	}

	return
}
