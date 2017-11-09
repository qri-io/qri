package cmd

import (
	"github.com/qri-io/qri/p2p"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:     "bootstrap",
	Aliases: []string{"bs"},
	Short:   "show or edit the list of qri bootstrap peers",
}

var bootstrapListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Show a configuration setting",
	Run: func(cmd *cobra.Command, args []string) {
		list := viper.GetStringSlice("bootstrap")
		for _, i := range list {
			PrintInfo("%s\n", i)
		}
	},
}

var boostrapAddCmd = &cobra.Command{
	Use:   "add",
	Short: "add peers to the bootstrap list",
	Run: func(cmd *cobra.Command, args []string) {
		_, err := p2p.ParseMultiaddrs(args)
		ExitIfErr(err)
		addrs := append(viper.GetStringSlice("bootstrap"), args...)
		// TODO - fix!!!
		WriteConfigFile(&Config{Bootstrap: addrs})
	},
}

var boostrapRmCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm"},
	Short:   "remove peers from the bootstrap list",
	Run: func(cmd *cobra.Command, args []string) {
		addrs := viper.GetStringSlice("bootstrap")
		for _, rm := range args {
			for i, adr := range addrs {
				if rm == adr {
					addrs = append(addrs[:i], addrs[i+1:]...)
				}
			}
		}
		// TODO - fix!!!
		WriteConfigFile(&Config{Bootstrap: addrs})
	},
}

func init() {
	bootstrapCmd.AddCommand(bootstrapListCmd)
	bootstrapCmd.AddCommand(boostrapAddCmd)
	bootstrapCmd.AddCommand(boostrapRmCmd)
	RootCmd.AddCommand(bootstrapCmd)
}
