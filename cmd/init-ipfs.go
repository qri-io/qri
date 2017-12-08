package cmd

import (
	"github.com/ipfs/go-datastore"
	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	initIpfsConfigFile string
)

// defaultDatasets is a hard-coded dataset added when a new qri repo is created
// this hash must always be available
var defaultDatasets = []datastore.Key{
	// fivethirtyeight comic characters
	datastore.NewKey("/ipfs/QmcqkHFA2LujZxY38dYZKmxsUstN4unk95azBjwEhwrnM6"),
}

// initCmd represents the init command
var initIpfsCmd = &cobra.Command{
	Use:   "init-ipfs",
	Short: "Initialize an ipfs repository",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		err := ipfs.InitRepo(viper.GetString(IpfsFsPath), initIpfsConfigFile, defaultDatasets)
		ExitIfErr(err)
	},
}

func init() {
	RootCmd.AddCommand(initIpfsCmd)
	initIpfsCmd.Flags().StringVarP(&initIpfsConfigFile, "config", "c", "", "config file for initialization")
}
