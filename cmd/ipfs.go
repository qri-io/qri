package cmd

import (
	ipfs "github.com/qri-io/castore/ipfs"
	"github.com/spf13/viper"
)

func GetIpfsDatastore() (*ipfs.Datastore, error) {
	return ipfs.NewDatastore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = viper.GetString(IpfsFsPath)
	})
}
