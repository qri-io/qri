package cmd

import (
	"fmt"

	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/qri/api"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	serverCmdPort string
	serverMemOnly bool
	serverOffline bool
	serverInit    bool
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

		if serverInit && !QRIRepoInitialized() {
			initCmd.Run(&cobra.Command{}, []string{})
		}

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
			cfg.PostP2POnlineHook = initializeDistributedAssets
		})
		ExitIfErr(err)

		err = s.Serve()
		ExitIfErr(err)
	},
}

// initializeDistributedAssets adds all distributed assets to the dataset
// by grabbing them from the network.
// eg.defaultDatasets, user profile photos & posters
func initializeDistributedAssets(node *p2p.QriNode) {
	cfg, err := ReadConfigFile()
	if err != nil || cfg.Initialized {
		return
	}

	if p, err := node.Repo.Profile(); err == nil {
		if pinner, ok := node.Repo.Store().(cafs.Pinner); ok {
			go func() {
				fmt.Println("pinning profile data")
				if p.Thumb.String() != "" {
					if err := pinner.Pin(p.Thumb, false); err != nil {
						fmt.Printf("error pinning thumb path: %s\n", err.Error())
					} else {
						fmt.Println("pinned thumb photo")
					}
				}
				if p.Profile.String() != "" {
					if err := pinner.Pin(p.Profile, false); err != nil {
						fmt.Printf("error pinning profile path: %s\n", err.Error())
					} else {
						fmt.Println("pinned profile photo photo")
					}
				}
				if p.Poster.String() != "" {
					if err := pinner.Pin(p.Poster, false); err != nil {
						fmt.Printf("error pinning poster path: %s\n", err.Error())
					} else {
						fmt.Println("pinned poster photo")
					}
				}
			}()
		}
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
	serverCmd.Flags().BoolVarP(&serverInit, "init", "", false, "initialize if necessary, reading options from enviornment variables")
	serverCmd.Flags().BoolVarP(&serverMemOnly, "mem-only", "", false, "run qri entirely in-memory, persisting nothing")
	serverCmd.Flags().BoolVarP(&serverOffline, "offline", "", false, "disable networking")
	RootCmd.AddCommand(serverCmd)
}
