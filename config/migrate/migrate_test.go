package migrate_test

import (
	"archive/zip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/config/migrate"
	"github.com/qri-io/qri/repo/buildrepo"
)

func TestOneToTwo(t *testing.T) {
	t.Skipf("This is an expensive test that downloads IPFS migration binaries. Should be run by hand")

	// setup a repo in the v1 arrangement
	dir, err := ioutil.TempDir("", "testOneToTwo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	repoPath := filepath.Join(dir, "qri")
	vOneIPFSPath := filepath.Join(dir, "ipfs")
	os.MkdirAll(repoPath, 0774)
	os.MkdirAll(vOneIPFSPath, 0774)
	os.Setenv("IPFS_PATH", vOneIPFSPath)

	input, err := ioutil.ReadFile("testdata/one_to_two/qri_config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	ioutil.WriteFile(filepath.Join(repoPath, "config.yaml"), input, 0774)
	unzipFile("testdata/one_to_two/empty_ipfs_repo_v0.4.21.zip", filepath.Join(dir, "ipfs"))

	cfg, err := config.ReadFromFile(filepath.Join(repoPath, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	// call OneToTwo
	if err := migrate.RunMigrations(ioes.NewDiscardIOStreams(), cfg, false); err != nil {
		t.Error(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buildrepo.New(ctx, repoPath, cfg)
}

func unzipFile(sourceZip, destDir string) {
	r, err := zip.OpenReader(sourceZip)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			panic(err)
		}
		defer rc.Close()

		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				panic(err)
			}
			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				panic(err)
			}
			_, err = io.Copy(outFile, rc)
			outFile.Close()
		}
	}
}
