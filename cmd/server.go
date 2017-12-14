package cmd

import (
	"fmt"
	"github.com/qri-io/qri/p2p"

	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/qri/api"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	serverCmdPort  string
	serverMemOnly  bool
	serverOffline  bool
	serverInitIpfs bool
)

// serverCmd represents the run command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start a qri server",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			r   repo.Repo
			err error
		)

		if serverMemOnly {
			// TODO - refine, adding better identity generation
			// or options for BYO user profile
			r, err = repo.NewMemRepo(
				&profile.Profile{
					Username: "mem user",
				},
				memfs.NewMapstore(),
				repo.MemPeers{},
				&analytics.Memstore{})
			ExitIfErr(err)
		} else {
			r = GetRepo(true)
		}

		s, err := api.New(r, func(cfg *api.Config) {
			cfg.Logger = log
			cfg.Port = serverCmdPort
			cfg.MemOnly = serverMemOnly
			cfg.Online = !serverOffline
			cfg.BoostrapAddrs = viper.GetStringSlice("bootstrap")
			cfg.PostP2POnlineHook = addDefaultDatasets
		})
		ExitIfErr(err)

		err = s.Serve()
		ExitIfErr(err)
	},
}

// Init sets up a repository with default datasets
func addDefaultDatasets(node *p2p.QriNode) {
	cfg, err := ReadConfigFile()
	if err != nil || cfg.Initialized {
		return
	}

	req := core.NewDatasetRequests(node.Repo, nil)

	for name, path := range cfg.DefaultDatasets {
		fmt.Printf("attempting to add default dataset: %s\n", path)
		res := &repo.DatasetRef{}
		err := req.AddDataset(&core.AddParams{
			Hash: path,
			Name: name,
		}, res)
		if err != nil {
			fmt.Printf("add dataset %s error: %s\n", path, err.Error())
			return
		}
		fmt.Printf("added default dataset: %s\n", path)
	}

	cfg.Initialized = true
	if err = WriteConfigFile(cfg); err != nil {
		fmt.Printf("error writing config file: %s", err.Error())
	}

	return
}

func init() {
	serverCmd.Flags().StringVarP(&serverCmdPort, "port", "p", api.DefaultPort, "port to start server on")
	// serverCmd.Flags().BoolVarP(&serverInitIpfs, "init-ipfs", "", false, "initialize a new default ipfs repo if empty")
	serverCmd.Flags().BoolVarP(&serverMemOnly, "mem-only", "", false, "run qri entirely in-memory, persisting nothing")
	serverCmd.Flags().BoolVarP(&serverOffline, "offline", "", false, "disable networking")
	RootCmd.AddCommand(serverCmd)
}
