package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/qipfs"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	testcfg "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/buildrepo"
)

// TempRepo manages a temporary repository for testing purposes, adding extra
// methods for testing convenience
type TempRepo struct {
	RootPath   string
	IPFSPath   string
	QriPath    string
	TestCrypto key.CryptoGenerator

	cfg                 *config.Config
	UseMockRemoteClient bool
}

// NewTempRepoFixedProfileID creates a temp repo that always uses the same
// PKI credentials
func NewTempRepoFixedProfileID(peername, prefix string) (r TempRepo, err error) {
	return newTempRepo(peername, prefix, NewTestCrypto())
}

// NewTempRepoUsingPeerInfo creates a temp repo using the given peerInfo
func NewTempRepoUsingPeerInfo(peerInfoNum int, peername, prefix string) (r TempRepo, err error) {
	crypto := NewTestCrypto()
	for i := 0; i < peerInfoNum; i++ {
		// TestCrypto uses a list of pre-generated private / public keys pairs, for performance
		// reasons and to make tests deterministic. Each time TestCrypt.GeneratePrivate... is
		// called, it will return the next peer info in this list. Most tests should always be
		// using different peer info, but may occassionally want them to match (to test conflicts).
		// This function can be used to skip a certain number of peer infos in order to get
		// a certain private key / profile ID that a test needs.
		_, _ = crypto.GeneratePrivateKeyAndPeerID()
	}
	return newTempRepo(peername, prefix, crypto)
}

// NewTempRepo constructs the test repo and initializes everything as cheaply
// as possible. This function is non-deterministic. Each successive call to
// TempRepo will use different PKI credentials
func NewTempRepo(peername, prefix string, g key.CryptoGenerator) (r TempRepo, err error) {
	return newTempRepo(peername, prefix, g)
}

func newTempRepo(peername, prefix string, g key.CryptoGenerator) (r TempRepo, err error) {
	RootPath, err := ioutil.TempDir("", prefix)
	if err != nil {
		return r, err
	}

	// Create directory for new Qri repo.
	QriPath := filepath.Join(RootPath, "qri")
	err = os.MkdirAll(QriPath, os.ModePerm)
	if err != nil {
		return r, err
	}
	// Create directory for new IPFS repo.
	IPFSPath := filepath.Join(QriPath, "ipfs")
	err = os.MkdirAll(IPFSPath, os.ModePerm)
	if err != nil {
		return r, err
	}
	// Build IPFS repo directory by unzipping an empty repo.
	err = g.GenerateEmptyIpfsRepo(IPFSPath, "")
	if err != nil {
		return r, err
	}

	// Create empty config.yaml into the test repo.
	cfg := testcfg.DefaultConfigForTesting().Copy()
	cfg.Profile.Peername = peername
	cfg.Profile.PrivKey, cfg.Profile.ID = g.GeneratePrivateKeyAndPeerID()
	cfg.SetPath(filepath.Join(QriPath, "config.yaml"))
	cfg.Filesystems = []qfs.Config{
		{Type: "ipfs", Config: map[string]interface{}{"path": IPFSPath}},
		{Type: "local"},
		{Type: "http"},
	}

	r = TempRepo{
		RootPath:   RootPath,
		IPFSPath:   IPFSPath,
		QriPath:    QriPath,
		TestCrypto: g,
		cfg:        cfg,
	}

	if err := r.WriteConfigFile(); err != nil {
		return r, err
	}

	return r, nil
}

// Repo constructs the repo for use in tests, the passed in context MUST be
// cancelled when finished. This repo creates it's own event bus
func (r *TempRepo) Repo(ctx context.Context) (repo.Repo, error) {
	return buildrepo.New(ctx, r.QriPath, r.cfg, func(o *buildrepo.Options) {
		o.Bus = event.NewBus(ctx)
	})
}

// Delete removes the test repo on disk.
func (r *TempRepo) Delete() {
	os.RemoveAll(r.RootPath)
}

// WriteConfigFile serializes the config file and writes it to the qri repository
func (r *TempRepo) WriteConfigFile() error {
	return r.cfg.WriteToFile(filepath.Join(r.QriPath, "config.yaml"))
}

// GetConfig returns the configuration for the test repo.
func (r *TempRepo) GetConfig() *config.Config {
	return r.cfg
}

// GetPathForDataset returns the path to where the index'th dataset is stored on CAFS.
func (r *TempRepo) GetPathForDataset(index int) (string, error) {
	dsRefs := filepath.Join(r.QriPath, "refs.fbs")

	data, err := ioutil.ReadFile(dsRefs)
	if err != nil {
		return "", err
	}

	refs, err := repo.UnmarshalRefsFlatbuffer(data)
	if err != nil {
		return "", err
	}

	// If dataset doesn't exist, return an empty string for the path.
	if len(refs) == 0 {
		return "", err
	}

	return refs[index].Path, nil
}

// ReadBodyFromIPFS reads the body of the dataset at the given keyPath stored
// in CAFS
func (r *TempRepo) ReadBodyFromIPFS(keyPath string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs, err := qipfs.NewFilesystem(ctx, map[string]interface{}{
		"online": false,
		"path":   r.IPFSPath,
	})

	if err != nil {
		return "", err
	}

	bodyFile, err := fs.Get(ctx, keyPath)
	if err != nil {
		return "", err
	}

	bodyBytes, err := ioutil.ReadAll(bodyFile)
	if err != nil {
		return "", err
	}

	done := gracefulShutdown(fs.(qfs.ReleasingFilesystem).Done())
	cancel()
	err = <-done
	return string(bodyBytes), err
}

// DatasetMarshalJSON reads the dataset head and marshals it as json.
func (r *TempRepo) DatasetMarshalJSON(ref string) (string, error) {
	ds, err := r.LoadDataset(ref)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(ds)
	if err != nil {
		return "", err
	}
	return string(data), err
}

// LoadDataset from the temp repository
func (r *TempRepo) LoadDataset(ref string) (*dataset.Dataset, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fs, err := qipfs.NewFilesystem(ctx, map[string]interface{}{
		"online": false,
		"path":   r.IPFSPath,
	})
	ds, err := dsfs.LoadDataset(ctx, fs, ref)
	if err != nil {
		return nil, err
	}
	done := gracefulShutdown(fs.(qfs.ReleasingFilesystem).Done())
	cancel()
	err = <-done
	return ds, err
}

// WriteRootFile writes a file string to the root directory of the temp repo
func (r *TempRepo) WriteRootFile(filename, data string) (path string, err error) {
	path = filepath.Join(r.RootPath, filename)
	err = ioutil.WriteFile(path, []byte(data), 0667)
	return path, err
}

// AddDatasets writes datasets to a temp repo
func (r *TempRepo) AddDatasets(ctx context.Context) (err error) {
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tempRepo, err := r.Repo(ctx)
	if err != nil {
		return err
	}

	for _, dsDirName := range datasets {
		tc, err := dstest.NewTestCaseFromDir(TestdataPath(dsDirName))
		if err != nil {
			return err
		}
		if _, err := createDataset(tempRepo, tc); err != nil {
			return fmt.Errorf("%s error creating dataset: %s", dsDirName, err.Error())
		}
	}

	cancel()
	<-tempRepo.Done()
	return tempRepo.DoneErr()
}

func gracefulShutdown(doneCh <-chan struct{}) chan error {
	waitForDone := make(chan error)
	go func() {
		select {
		case <-time.NewTimer(time.Second).C:
			waitForDone <- fmt.Errorf("shutdown didn't send on 'done' channel within 1 second of context cancellation")
		case <-doneCh:
			waitForDone <- nil
		}
	}()
	return waitForDone
}
