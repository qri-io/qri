package cmd

import (
	"context"
	"fmt"
	// "io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	// bstore "github.com/ipfs/go-ipfs/blocks/blockstore"
	blockservice "github.com/ipfs/go-ipfs/blockservice"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreunix"
	dag "github.com/ipfs/go-ipfs/merkledag"
	// dagtest "github.com/ipfs/go-ipfs/merkledag/test"
	files "github.com/ipfs/go-ipfs/commands/files"
	path "github.com/ipfs/go-ipfs/path"
	repo "github.com/ipfs/go-ipfs/repo"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"
	tar "github.com/ipfs/go-ipfs/thirdparty/tar"
	uarchive "github.com/ipfs/go-ipfs/unixfs/archive"
	"gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
)

var (
	// this machine's local ipfs filestore repo
	localRepo repo.Repo
	// networkless ipfs node
	node *core.IpfsNode
)

func init() {
	var err error
	ctx := context.Background()
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	localRepo, err = fsrepo.Open("~/ipfs")
	if err != nil {
		fmt.Printf("error opening local filestore ipfs repository: %s\n", err.Error())
		return
	}

	cfg := &core.BuildCfg{
		Repo:   localRepo,
		Online: false,
	}

	node, err = core.NewNode(ctx, cfg)
	if err != nil {
		fmt.Printf("error creating networkless ipfs node: %s\n", err.Error())
		return
	}
}

func getKey(key datastore.Key) ([]byte, error) {
	p := path.Path(key.String())
	ctx := context.Background()
	dn, err := core.Resolve(ctx, node.Namesys, node.Resolver, p)
	if err != nil {
		fmt.Println("resolver error")
		return nil, err
	}

	// switch dn := dn.(type) {
	// case *dag.ProtoNode:
	// 	size, err := dn.Size()
	// 	if err != nil {
	// 		res.SetError(err, cmds.ErrNormal)
	// 		return
	// 	}

	// 	res.SetLength(size)
	// case *dag.RawNode:
	// 	res.SetLength(uint64(len(dn.RawData())))
	// default:
	// 	res.SetError(fmt.Errorf("'ipfs get' only supports unixfs nodes"), cmds.ErrNormal)
	// 	return
	// }

	rdr, err := uarchive.DagArchive(ctx, dn, p.String(), node.DAG, false, 0)
	if err != nil {
		return nil, err
	}

	fp := filepath.Join("/tmp", key.BaseNamespace())

	e := tar.Extractor{
		Path:     fp,
		Progress: func(int64) int64 { return 0 },
	}
	if err := e.Extract(rdr); err != nil {
		return nil, err
	}

	return ioutil.ReadFile(fp)
}

func addAndPinFile(filename string, data []byte) (hash string, err error) {
	if localRepo == nil {
		return "", fmt.Errorf("no local ipfs repo to write to")
	}
	if node == nil {
		return "", fmt.Errorf("networkless ipfs node isn't initialized")
	}

	ctx := context.Background()
	bserv := blockservice.New(node.Blockstore, node.Exchange)
	dagserv := dag.NewDAGService(bserv)

	fileAdder, err := coreunix.NewAdder(ctx, node.Pinning, node.Blockstore, dagserv)
	if err != nil {
		return
	}

	path := filepath.Join("/tmp", time.Now().String())

	if err = ioutil.WriteFile(path, data, os.ModePerm); err != nil {
		return
	}

	fi, err := os.Stat(path)
	if err != nil {
		return
	}

	rfile, err := files.NewSerialFile(filename, path, false, fi)
	if err != nil {
		return
	}

	outChan := make(chan interface{}, 8)
	defer close(outChan)

	fileAdder.Out = outChan

	if err = fileAdder.AddFile(rfile); err != nil {
		return
	}

	if _, err = fileAdder.Finalize(); err != nil {
		return
	}

	if err = fileAdder.PinRoot(); err != nil {
		return
	}

	for {
		select {
		case out, ok := <-outChan:
			if ok {
				output := out.(*coreunix.AddedObject)
				if len(output.Hash) > 0 {
					hash = output.Hash
					return
				}
			}
		}
	}

	err = fmt.Errorf("something's gone horribly wrong")
	return
}
