package cmd

import (
	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/qri/api"
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
		if serverInitIpfs {
			err := initRepoIfEmpty(viper.GetString(IpfsFsPath), "")
			ExitIfErr(err)
		}

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
		})
		ExitIfErr(err)

		err = s.Serve()
		ExitIfErr(err)
	},
}

// Init sets up a repository with sensible defaults
func addDefaultDatasets() error {
	// req, err := DatasetRequests(true)
	// if err != nil {
	// 	return err
	// }

	// TODO - restore
	// for name, ds := range defaultDatasets {
	// 	fmt.Printf("attempting to add default dataset: %s\n", ds.String())
	// 	res := &repo.DatasetRef{}
	// 	err := req.AddDataset(&core.AddParams{
	// 		Hash: ds.String(),
	// 		Name: name,
	// 	}, res)
	// 	if err != nil {
	// 		fmt.Printf("add dataset %s error: %s\n", ds.String(), err.Error())
	// 		return err
	// 	}
	// 	fmt.Printf("added default dataset: %s\n", ds.String())
	// }

	return nil
}

func init() {
	serverCmd.Flags().StringVarP(&serverCmdPort, "port", "p", api.DefaultPort, "port to start server on")
	serverCmd.Flags().BoolVarP(&serverInitIpfs, "init-ipfs", "", false, "initialize a new default ipfs repo if empty")
	serverCmd.Flags().BoolVarP(&serverMemOnly, "mem-only", "", false, "run qri entirely in-memory, persisting nothing")
	serverCmd.Flags().BoolVarP(&serverOffline, "offline", "", false, "disable networking")
	RootCmd.AddCommand(serverCmd)
}
