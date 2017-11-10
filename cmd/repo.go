package cmd

import (
	"strings"

	"github.com/ipfs/go-datastore"
	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
	"github.com/spf13/viper"
)

var r repo.Repo

func GetRepo(online bool) repo.Repo {
	if r != nil {
		return r
	}

	fs := GetIpfsFilestore(online)
	id := ""
	if fs.Node().PeerHost != nil {
		id = fs.Node().PeerHost.ID().Pretty()
	}

	r, err := fs_repo.NewRepo(fs, viper.GetString(QriRepoPath), id)
	ExitIfErr(err)
	return r
}

func GetIpfsFilestore(online bool) *ipfs.Filestore {
	fs, err := ipfs.NewFilestore(func(cfg *ipfs.StoreCfg) {
		cfg.FsRepoPath = viper.GetString(IpfsFsPath)
		cfg.Online = online
	})
	ExitIfErr(err)
	return fs
}

// TODO - consider moving this to the dsfs package
func Ref(input string) (refType, ref string) {
	if strings.HasPrefix(input, "/ipfs/") || strings.HasSuffix(input, dsfs.PackageFileDataset.Filename()) {
		return "path", cleanHash(input)
	} else if len(input) == 46 && !strings.Contains(input, "_") {
		// 46 is the current length of a base58-encoded hash on ipfs
		return "path", cleanHash(input)
	}

	return "name", input
}

func cleanHash(in string) string {
	if !strings.HasPrefix(in, "/ipfs/") {
		in = datastore.NewKey("/ipfs/").ChildString(in).String()
	}
	if !strings.HasSuffix(in, dsfs.PackageFileDataset.String()) {
		in = datastore.NewKey(in).ChildString(dsfs.PackageFileDataset.String()).String()
	}
	return in
}
