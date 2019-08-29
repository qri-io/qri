package actions

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo/gen"
)

// Setup provisions a new qri instance
func Setup(repoPath, cfgPath string, cfg *config.Config, register bool) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %s", err.Error())
	}

	// TODO (b5) - we'll need to add a choose-password-for-qri-cloud to do
	// signup for this
	// if register && cfg.Registry != nil {
	// 	pro, err := profile.NewProfile(cfg.Profile)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	reg := regclient.NewClient(&regclient.Config{
	// 		Location: cfg.Registry.Location,
	// 	})

	// 	if _, err := reg.PutProfile(pro, pro.PrivKey); err != nil {
	// 		return err
	// 	}
	// }

	if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
		return fmt.Errorf("error creating qri repo directory: %s, path: %s", err.Error(), repoPath)
	}

	if err := cfg.WriteToFile(cfgPath); err != nil {
		return fmt.Errorf("error writing config: %s", err.Error())
	}
	return nil
}

// InitIPFS initializes an IPFS repo
func InitIPFS(path string, cfgData []byte, g gen.CryptoGenerator) error {
	tmpIPFSConfigPath := ""
	if cfgData != nil {
		// TODO - remove this temp file & instead adjust ipfs.InitRepo to accept an io.Reader
		tmpIPFSConfigPath = filepath.Join(os.TempDir(), "ipfs_init_config")

		if err := ioutil.WriteFile(tmpIPFSConfigPath, cfgData, os.ModePerm); err != nil {
			return err
		}

		defer func() {
			os.Remove(tmpIPFSConfigPath)
		}()
	}

	if err := g.GenerateEmptyIpfsRepo(path, tmpIPFSConfigPath); err != nil {
		if !strings.Contains(err.Error(), "already") {
			return fmt.Errorf("error creating IPFS repo: %s", err.Error())
		}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("no IPFS repo exists at %s, things aren't going to work properly", path)
	}
	return nil
}

// Teardown reverses the setup process
func Teardown(repoPath string, cfg *config.Config) error {

	return os.RemoveAll(repoPath)
}
