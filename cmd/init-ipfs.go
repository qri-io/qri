package cmd

import (
	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	initIpfsConfigFile string
)

// initCmd represents the init command
var initIpfsCmd = &cobra.Command{
	Use:   "init-ipfs",
	Short: "Initialize an ipfs repository",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		err := ipfs.InitRepo(viper.GetString(IpfsFsPath), initIpfsConfigFile)
		ExitIfErr(err)
	},
}

func init() {
	RootCmd.AddCommand(initIpfsCmd)
	initIpfsCmd.Flags().StringVarP(&initIpfsConfigFile, "config", "c", "", "config file for initialization")
}
