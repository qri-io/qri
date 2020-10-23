package dsfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/qipfs"
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

	mem := qfs.NewMemFS()

	cases := []struct {
		fs   qfs.Filesystem
		path string
		pf   PackageFile
		out  string
	}{
		{ipfs, "/ipfs/foo", PackageFileDataset, "/ipfs/foo/dataset.json"},
		{ipfs, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M", PackageFileDataset, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
		{ipfs, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json", PackageFileDataset, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
		{ipfs, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json", PackageFileMeta, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/meta.json"},
		{ipfs, "QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M", PackageFileDataset, "/ipfs/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},

		{mem, "/mem/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M", PackageFileDataset, "/mem/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
		{mem, "/mem/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json", PackageFileDataset, "/mem/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json"},
		{mem, "/mem/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/dataset.json", PackageFileMeta, "/mem/QmZfwmhbcgSDGqGaoMMYx8jxBGauZw75zPjnZAyfwPso7M/meta.json"},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			got := PackageFilepath(c.fs, c.path, c.pf)
			if got != c.out {
				t.Errorf("result mismatch. expected: '%s', got: '%s'", c.path, c.pf)
			}
		})
	}
}

func makeTestIPFSRepo(ctx context.Context, path string) (fs *qipfs.Filestore, destroy func(), err error) {
	if path == "" {
		tmp, err := ioutil.TempDir("", "temp-ipfs-repo")
		if err != nil {
			panic(err)
		}
		path = filepath.Join(tmp, ".ipfs")
	}
	err = qipfs.InitRepo(path, "")
	if err != nil {
		return
	}

	qfsFilestore, err := qipfs.NewFilesystem(ctx, map[string]interface{}{"path": path})
	if err != nil {
		return
	}

	fs, ok := qfsFilestore.(*qipfs.Filestore)
	if !ok {
		return nil, nil, fmt.Errorf("created filestore is not of type ipfs")
	}

	destroy = func() {
		os.RemoveAll(path)
	}

	return
}
