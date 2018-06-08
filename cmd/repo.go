package cmd

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"strings"
	"sync"
	"time"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/registry/regclient"
)

var (
	repository  repo.Repo
	rpcClient   *rpc.Client
	rpcConnOnce sync.Once
)

func init() {
	// use a short timeout for registry clients to keep cli working
	// if registry interaction fails it's not the end of the world
	regclient.HTTPClient = &http.Client{
		Timeout: time.Second * 4,
	}
}

func rpcConn() *rpc.Client {
	onceBody := func() {
		// TODO - replace by forcing a default core.Config to exist
		addr := ":2504"
		if core.Config != nil {
			addr = fmt.Sprintf(":%d", core.Config.RPC.Port)
		}
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return
		}
		rpcClient = rpc.NewClient(conn)
	}
	rpcConnOnce.Do(onceBody)
	return rpcClient
}

func getRepo(online bool) repo.Repo {
	if repository != nil {
		return repository
	}

	if !QRIRepoInitialized() {
		ErrExit(fmt.Errorf("no qri repo found, please run `qri setup`"))
	}

	fs := getIpfsFilestore(online)
	pro, err := profile.NewProfile(core.Config.Profile)
	ExitIfErr(err)

	var rc *regclient.Client
	if core.Config.Registry != nil && core.Config.Registry.Location != "" {
		rc = regclient.NewClient(&regclient.Config{
			Location: core.Config.Registry.Location,
		})
	}

	r, err := fsrepo.NewRepo(fs, pro, rc, QriRepoPath)
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

func requireNotRPC(cmdName string) {
	if core.Config.RPC.Enabled {
		if cli := rpcConn(); cli != nil {
			err := fmt.Errorf(`sorry, we can't run the '%s' command while 'qri connect' is running
we know this is super irritating, and it'll be fixed in the future. 
In the meantime please close qri and re-run this command`, cmdName)
			ErrExit(err)
		}
	}
}

func datasetRequests(online bool) (*core.DatasetRequests, error) {
	if cli := rpcConn(); cli != nil {
		return core.NewDatasetRequests(nil, cli), nil
	}

	if !online {
		// TODO - make this not terrible
		r, cli, err := repoOrClient(online)
		if err != nil {
			return nil, err
		}
		return core.NewDatasetRequests(r, cli), nil
	}

	n, err := qriNode(online)
	if err != nil {
		return nil, err
	}

	req := core.NewDatasetRequests(n.Repo, nil)
	req.Node = n
	return req, nil
}

func registryRequests(online bool) (*core.RegistryRequests, error) {
	if cli := rpcConn(); cli != nil {
		return core.NewRegistryRequests(nil, cli), nil
	}

	if !online {
		// TODO - make this not terrible
		r, cli, err := repoOrClient(online)
		if err != nil {
			return nil, err
		}
		return core.NewRegistryRequests(r, cli), nil
	}

	n, err := qriNode(online)
	if err != nil {
		return nil, err
	}

	return core.NewRegistryRequests(n.Repo, nil), nil
}

func renderRequests(online bool) (*core.RenderRequests, error) {
	if cli := rpcConn(); cli != nil {
		return core.NewRenderRequests(nil, cli), nil
	}

	if !online {
		// TODO - make this not terrible
		r, cli, err := repoOrClient(online)
		if err != nil {
			return nil, err
		}
		return core.NewRenderRequests(r, cli), nil
	}

	n, err := qriNode(online)
	if err != nil {
		return nil, err
	}

	req := core.NewRenderRequests(n.Repo, nil)
	return req, nil
}

func profileRequests(online bool) (*core.ProfileRequests, error) {
	r, cli, err := repoOrClient(online)
	if err != nil {
		return nil, err
	}
	return core.NewProfileRequests(r, cli), nil
}

// func searchRequests(online bool) (*core.SearchRequests, error) {
// 	r, cli, err := repoOrClient(online)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return core.NewSearchRequests(r, cli), nil
// }

func historyRequests(online bool) (*core.HistoryRequests, error) {
	if cli := rpcConn(); cli != nil {
		return core.NewHistoryRequests(nil, cli), nil
	}

	if !online {
		// TODO - make this not terrible
		r, cli, err := repoOrClient(online)
		if err != nil {
			return nil, err
		}
		return core.NewHistoryRequests(r, cli), nil
	}

	n, err := qriNode(online)
	if err != nil {
		return nil, err
	}

	req := core.NewHistoryRequests(n.Repo, nil)
	req.Node = n
	return req, nil
}

func peerRequests(online bool) (*core.PeerRequests, error) {
	if cli := rpcConn(); cli != nil {
		return core.NewPeerRequests(nil, cli), nil
	}

	node, err := qriNode(online)
	if err != nil {
		return nil, err
	}
	return core.NewPeerRequests(node, nil), nil
}

func repoOrClient(online bool) (repo.Repo, *rpc.Client, error) {
	if repository != nil {
		return repository, nil, nil
	} else if cli := rpcConn(); cli != nil {
		return nil, cli, nil
	}

	if fs, err := ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = IpfsFsPath
		cfg.Online = online
	}); err == nil {
		pro, err := profile.NewProfile(core.Config.Profile)
		ExitIfErr(err)

		var rc *regclient.Client
		if core.Config.Registry != nil && core.Config.Registry.Location != "" {
			rc = regclient.NewClient(&regclient.Config{Location: core.Config.Registry.Location})
		}

		r, err := fsrepo.NewRepo(fs, pro, rc, QriRepoPath)
		ExitIfErr(err)

		return r, nil, err

	} else if strings.Contains(err.Error(), "lock") {
		conn := rpcConn()
		if conn != nil {
			return nil, conn, nil
		}
		return nil, nil, err
	} else {
		return nil, nil, err
	}

	return nil, nil, fmt.Errorf("badbadnotgood")
}

func qriNode(online bool) (node *p2p.QriNode, err error) {
	var (
		r  repo.Repo
		fs *ipfs.Filestore
		rc *regclient.Client
	)

	fs, err = ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = IpfsFsPath
		cfg.Online = online
	})

	if err != nil {
		return
	}

	pro, err := profile.NewProfile(core.Config.Profile)
	ExitIfErr(err)

	if core.Config.Registry != nil && core.Config.Registry.Location != "" {
		rc = regclient.NewClient(&regclient.Config{Location: core.Config.Registry.Location})
	}
	printInfo("%v", rc)

	r, err = fsrepo.NewRepo(fs, pro, rc, QriRepoPath)
	if err != nil {
		return
	}

	node, err = p2p.NewQriNode(r, func(c *config.P2P) {
		c.Enabled = online
		c.QriBootstrapAddrs = core.Config.P2P.QriBootstrapAddrs
	})
	if err != nil {
		return
	}

	return
}
