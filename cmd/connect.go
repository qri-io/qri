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
	connectCmdPort    string
	connectCmdRPCPort string
	connectMemOnly    bool
	connectOffline    bool
	connectSetup      bool
)

// connectCmd represents the run command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect to the distributed web, start a local API server",
	Long: `
While it’s not totally accurate, connect is like starting a server. running 
connect will start a process and stay there until you exit the process 
(ctrl+c from the terminal, or killing the process using tools like activity 
monitor on the mac, or the aptly-named “kill” command). Connect does three main 
things:
- Connect to the qri distributed network
- Connect to IPFS
- Start a local API server

When you run connect you are connecting to the distributed web, interacting with
peers & swapping data.

The default port for the local API server is 2503. We call port 2503,
“the qri port”. It’s a good port, lots of cool numbers in there. Some might even
call it a “prime” port number.`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			r   repo.Repo
			err error
		)

		if connectSetup && !QRIRepoInitialized() {
			setupCmd.Run(&cobra.Command{}, []string{})
		}

		if connectMemOnly {
			// TODO - refine, adding better identity generation
			// or options for BYO user profile
			r, err = repo.NewMemRepo(
				&profile.Profile{
					Peername: "mem user",
				},
				memfs.NewMapstore(),
				repo.MemPeers{},
				&analytics.Memstore{})
			ExitIfErr(err)
		} else {
			r = getRepo(true)
		}

		s, err := api.New(r, func(cfg *api.Config) {
			cfg.Logger = log
			cfg.Port = connectCmdPort
			cfg.RPCPort = connectCmdRPCPort
			cfg.MemOnly = connectMemOnly
			cfg.Online = !connectOffline
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
	cfg, err := readConfigFile()
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

	for _, refstr := range cfg.DefaultDatasets {
		fmt.Printf("attempting to add default dataset: %s\n", refstr)
		ref, err := repo.ParseDatasetRef(refstr)
		if err != nil {
			fmt.Println("error parsing dataset reference: '%s': %s", refstr, err.Error())
			continue
		}
		res := &repo.DatasetRef{}
		err = req.Add(ref, res)
		if err != nil {
			fmt.Printf("add dataset %s error: %s\n", refstr, err.Error())
			return
		}
		fmt.Printf("added default dataset: %s\n", refstr)
	}

	cfg.Initialized = true
	if err = writeConfigFile(cfg); err != nil {
		fmt.Printf("error writing config file: %s", err.Error())
	}

	return
}

func init() {
	connectCmd.Flags().StringVarP(&connectCmdPort, "api-port", "", api.DefaultPort, "port to start api on")
	connectCmd.Flags().StringVarP(&connectCmdRPCPort, "rpc-port", "", api.DefaultRPCPort, "port to start rpc listener on")
	connectCmd.Flags().BoolVarP(&connectSetup, "setup", "", false, "run setup if necessary, reading options from enviornment variables")
	connectCmd.Flags().BoolVarP(&connectMemOnly, "mem-only", "", false, "run qri entirely in-memory, persisting nothing")
	connectCmd.Flags().BoolVarP(&connectOffline, "offline", "", false, "disable networking")
	RootCmd.AddCommand(connectCmd)
}
