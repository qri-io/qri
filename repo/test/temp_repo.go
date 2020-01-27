package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/ioes"
	ipfs_filestore "github.com/qri-io/qfs/cafs/ipfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/buildrepo"
	"github.com/qri-io/qri/repo/gen"
)

// TempRepo manages a temporary repository for testing purposes, adding extra
// methods for testing convenience
type TempRepo struct {
	RootPath            string
	IPFSPath            string
	QriPath             string
	TestCrypto          gen.CryptoGenerator
	Streams             ioes.IOStreams
	cfg                 *config.Config
	repo                repo.Repo
	UseMockRemoteClient bool
}

// NewTempRepo constructs the test repo and initializes everything as cheaply as possible.
func NewTempRepo(peername, prefix string) (r TempRepo, err error) {
	RootPath, err := ioutil.TempDir("", prefix)
	if err != nil {
		return r, err
	}

	// Create directory for new IPFS repo.
	IPFSPath := filepath.Join(RootPath, "ipfs")
	err = os.MkdirAll(IPFSPath, os.ModePerm)
	if err != nil {
		return r, err
	}
	// Build IPFS repo directory by unzipping an empty repo.
	err = defaultCryptoGenerator.GenerateEmptyIpfsRepo(IPFSPath, "")
	if err != nil {
		return r, err
	}
	// Create directory for new Qri repo.
	QriPath := filepath.Join(RootPath, "qri")
	err = os.MkdirAll(QriPath, os.ModePerm)
	if err != nil {
		return r, err
	}

	// Create empty config.yaml into the test repo.
	cfg := config.DefaultConfigForTesting().Copy()
	cfg.Profile.Peername = peername
	cfg.Profile.PrivKey, cfg.Profile.ID = defaultCryptoGenerator.GeneratePrivateKeyAndPeerID()
	cfg.Store.Path = IPFSPath

	r = TempRepo{
		RootPath:   RootPath,
		IPFSPath:   IPFSPath,
		QriPath:    QriPath,
		TestCrypto: defaultCryptoGenerator,
		cfg:        cfg,
	}
	if err := r.WriteConfigFile(); err != nil {
		return r, err
	}
	return r, nil
}

// Repo accesses the actual repo, building one if it doesn't already exist
func (r *TempRepo) Repo() (repo.Repo, error) {
	if r.repo == nil {
		var err error
		if r.repo, err = buildrepo.New(context.TODO(), r.RootPath, r.cfg); err != nil {
			return nil, err
		}
	}
	return r.repo, nil
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

// GetOutput returns the output from the previously executed command.
func (r *TempRepo) GetOutput() string {
	buffer, ok := r.Streams.Out.(*bytes.Buffer)
	if ok {
		return buffer.String()
	}
	return ""
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
	ctx := context.Background()
	fs, err := ipfs_filestore.NewFilestore(func(cfg *ipfs_filestore.StoreCfg) {
		cfg.Online = false
		cfg.FsRepoPath = r.IPFSPath
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

	return string(bodyBytes), nil
}

// DatasetMarshalJSON reads the dataset head and marshals it as json.
func (r *TempRepo) DatasetMarshalJSON(ref string) (string, error) {
	ctx := context.Background()
	fs, err := ipfs_filestore.NewFilestore(func(cfg *ipfs_filestore.StoreCfg) {
		cfg.Online = false
		cfg.FsRepoPath = r.IPFSPath
	})
	ds, err := dsfs.LoadDataset(ctx, fs, ref)
	if err != nil {
		return "", err
	}
	bytes, err := json.Marshal(ds)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
