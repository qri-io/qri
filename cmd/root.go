package cmd

import (
	"flag"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/core"
	"github.com/qri-io/qri/repo"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

const (
	QriRepoPath = "QriRepoPath"
	IpfsFsPath  = "IpfsFsPath"
)

// global pagination variables
var (
	pageNum  int
	pageSize int
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "qri",
	Short: "qri.io command line client",
	Long: `this is a very early tool for working with datasets on the distributed web.
	At the moment it's a bit an experiment.

	Email brendan with any questions:
	sparkle_pony_2000@qri.io`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		PrintErr(err)
		os.Exit(-1)
	}
}

func init() {
	flag.Parse()
	cobra.OnInitialize(initialize)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $QRI_PATH/config.json)")
	RootCmd.PersistentFlags().BoolVarP(&noColor, "no-color", "c", false, "disable colorized output")
}

func initialize() {
	created := initConfig()
	if created {
		go addDefaultDatasets()
	}
}

// defaultDatasets is a hard-coded dataset added when a new qri repo is created
// these hashes should always/highly available
var defaultDatasets = map[string]datastore.Key{
	// fivethirtyeight comic characters
	"comic_characters": datastore.NewKey("/ipfs/QmcqkHFA2LujZxY38dYZKmxsUstN4unk95azBjwEhwrnM6/dataset.json"),
}

// Init sets up a repository with sensible defaults
func addDefaultDatasets() error {
	req, err := DatasetRequests(true)
	if err != nil {
		return err
	}

	for name, ds := range defaultDatasets {
		fmt.Printf("attempting to add default dataset: %s\n", ds.String())
		res := &repo.DatasetRef{}
		err := req.AddDataset(&core.AddParams{
			Hash: ds.String(),
			Name: name,
		}, res)
		if err != nil {
			fmt.Printf("add dataset %s error: %s\n", ds.String(), err.Error())
			return err
		}
		fmt.Printf("added default dataset: %s\n", ds.String())
	}

	return nil
}
