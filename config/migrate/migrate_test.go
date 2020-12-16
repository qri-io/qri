package migrate_test

import (
	"archive/zip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	if err := migrate.RunMigrations(ioes.NewDiscardIOStreams(), cfg, func() bool { return true }, false); err != nil {
		t.Error(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buildrepo.New(ctx, repoPath, cfg)
}

func TestTwoToThree(t *testing.T) {
	// setup a repo in the v2 arrangement
	dir, err := ioutil.TempDir("", "testTwoToThree")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	repoPath := filepath.Join(dir, "qri")
	os.MkdirAll(repoPath, 0774)

	input, err := ioutil.ReadFile("testdata/two_to_three/qri_config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	ioutil.WriteFile(filepath.Join(repoPath, "config.yaml"), input, 0774)

	cfg, err := config.ReadFromFile(filepath.Join(repoPath, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	// call TwoToThree
	if err := migrate.RunMigrations(ioes.NewStdIOStreams(), cfg, func() bool { return true }, false); err != nil {
		t.Error(err)
	}

	expect := []string{
		"/ip4/1.2.3.4/tcp/4001/ipfs/QmTestPersistingManuallyAddedBootstrappers",            // should persist unknown
		"/ip4/35.231.230.13/tcp/4001/ipfs/QmdpGkbqDYRPCcwLYnEm8oYGz2G9aUZn9WwPjqvqw3XUAc",  // red
		"/ip4/34.75.40.163/tcp/4001/ipfs/QmTRqTLbKndFC2rp6VzpyApxHCLrFV35setF1DQZaRWPVf",   // orange
		"/ip4/35.237.172.74/tcp/4001/ipfs/QmegNYmwHUQFc3v3eemsYUVf3WiSg4RcMrh3hovA5LncJ2",  // yellow
		"/ip4/35.231.155.111/tcp/4001/ipfs/QmessbA6uGLJ7HTwbUJ2niE49WbdPfzi27tdYXdAaGRB4G", // green
		"/ip4/35.237.232.64/tcp/4001/ipfs/Qmc353gHY5Wx5iHKHPYj3QDqHP4hVA1MpoSsT6hwSyVx3r",  // blue
		"/ip4/35.185.20.61/tcp/4001/ipfs/QmT9YHJF2YkysLqWhhiVTL5526VFtavic3bVueF9rCsjVi",   // indigo
		"/ip4/35.231.246.50/tcp/4001/ipfs/QmQS2ryqZrjJtPKDy9VTkdPwdUSpTi1TdpGUaqAVwfxcNh",  // violet
	}

	if diff := cmp.Diff(cfg.P2P.QriBootstrapAddrs, expect); diff != "" {
		t.Errorf("config.p2p.QriBootstrapAddrs result mismatch. (-want +got):%s\n", diff)
	}
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
