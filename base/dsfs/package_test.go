package dsfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/qfs/cafs"
	ipfsfs "github.com/qri-io/qfs/cafs/ipfs"
	"golang.org/x/net/context"
)

func TestPackageFilepath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ipfs, destroy, err := makeTestIPFSRepo(ctx, "")
	if err != nil {
		t.Errorf("error creating IPFS test repo: %s", err.Error())
		return
	}
	defer destroy()

	mem := cafs.NewMapstore()

	cases := []struct {
		store cafs.Filestore
		path  string
		pf    PackageFile
		out   string
	}{
		{ipfs, "/ipfs/foo", PackageFileDataset, "/ipfs/foo/dataset.json"},
		{ipfs, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M", PackageFileDataset, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
		{ipfs, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json", PackageFileDataset, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
		{ipfs, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json", PackageFileMeta, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/meta.json"},
		{ipfs, "QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M", PackageFileDataset, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},

		{mem, "/map/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M", PackageFileDataset, "/map/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
		{mem, "/map/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json", PackageFileDataset, "/map/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
		{mem, "/map/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json", PackageFileMeta, "/map/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/meta.json"},
	}

	for i, c := range cases {
		got := PackageFilepath(c.store, c.path, c.pf)
		if got != c.out {
			t.Errorf("case %d result mismatch. expected: '%s', got: '%s'", i, c.path, c.pf)
			continue
		}
	}
}

func makeTestIPFSRepo(ctx context.Context, path string) (fs *ipfsfs.Filestore, destroy func(), err error) {
	if path == "" {
		tmp, err := ioutil.TempDir("", "temp-ipfs-repo")
		if err != nil {
			panic(err)
		}
		path = filepath.Join(tmp, ".ipfs")
	}
	err = ipfsfs.InitRepo(path, "")
	if err != nil {
		return
	}

	qfsFilestore, err := ipfsfs.NewFS(ctx, map[string]interface{}{"fsRepoPath": path})
	if err != nil {
		return
	}

	fs, ok := qfsFilestore.(*ipfsfs.Filestore)
	if !ok {
		return nil, nil, fmt.Errorf("created filestore is not of type ipfs")
	}

	destroy = func() {
		os.RemoveAll(path)
	}

	return
}
