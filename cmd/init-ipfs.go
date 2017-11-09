package cmd

import (
	"flag"
	"fmt"
	"path/filepath"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/spf13/cobra"
)

var (
	initIpfsConfigFile string
	// initMetaFile   string
	// initName       string
	// initPassive    bool
	// initRescursive bool
)

// initCmd represents the init command
var initIpfsCmd = &cobra.Command{
	Use:   "init-ipfs",
	Short: "Initialize an ipfs repository",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if initFile == "" {
			ErrExit(fmt.Errorf("please provide a file argument"))
		}

		path, err := filepath.Abs(initFile)
		ExitIfErr(err)

		err = ipfs.InitRepo(path, initIpfsConfigFile)
		ExitIfErr(err)
	},
}

func init() {
	flag.Parse()
	RootCmd.AddCommand(initIpfsCmd)
	initIpfsCmd.Flags().StringVarP(&initIpfsConfigFile, "config", "c", "", "config file for initialization")
}
