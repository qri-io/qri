package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/qri/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	config "gx/ipfs/QmViBzgruNUoLNBnXcx8YWbDNwV8MNGEGKkLo6JGetygdw/go-ipfs/repo/config"
)

var (
	setupOverwrite      bool
	setupIPFS           bool
	setupIPFSConfigFile string
	setupIdentityData   string
	setupProfileData    string
	setupDatasetsData   string
	setupBootstrapData  string
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initialize qri and IPFS repositories, provision a new qri ID",
	Long: `
Usage:
	qri setup [--no-ipfs]

Setup is the first command you run to get a fresh install of qri. If you’ve 
never run qri before, you’ll need to run setup before you can do anything. 

Setup does a few things:
- create a qri repository to keep all of your data
- provisions a new qri ID
- create an IPFS repository if one doesn’t exist

This command is automatically run if you invoke any qri command without first 
running setup. If setup has already been run, by default qri won’t let you 
overwrite this info.`,
	Run: func(cmd *cobra.Command, args []string) {
		var cfgData []byte

		if QRIRepoInitialized() && !setupOverwrite {
			// use --overwrite to overwrite this repo, erasing all data and deleting your account for good
			// this is usually a terrible idea
			ErrExit(fmt.Errorf("repo already initialized"))
		}
		fmt.Println("setupializing qri repo")

		envVars := map[string]*string{
			"QRI_INIT_IDENTITY_DATA":  &setupIdentityData,
			"QRI_INIT_PROFILE_DATA":   &setupProfileData,
			"QRI_INIT_DATASETS_DATA":  &setupDatasetsData,
			"QRI_INIT_BOOTSTRAP_DATA": &setupBootstrapData,
		}
		mapEnvVars(envVars)

		// if cfgFile is specified, override
		if cfgFile != "" {
			f, err := os.Open(cfgFile)
			ExitIfErr(err)
			cfgData, err = ioutil.ReadAll(f)
			ExitIfErr(err)
		} else {
			cfgData = defaultCfgBytes()
		}

		cfg := &Config{}
		err := yaml.Unmarshal(cfgData, cfg)
		ExitIfErr(err)

		err = cfg.ensurePrivateKey()
		ExitIfErr(err)

		if setupDatasetsData != "" {
			err = readAtFile(&setupDatasetsData)
			ExitIfErr(err)

			datasets := map[string]string{}
			err = json.Unmarshal([]byte(setupDatasetsData), &datasets)
			ExitIfErr(err)

			cfg.DefaultDatasets = datasets
		}

		if setupBootstrapData != "" {
			err = readAtFile(&setupBootstrapData)
			ExitIfErr(err)

			bootstrap := []string{}
			err = json.Unmarshal([]byte(setupBootstrapData), &bootstrap)
			ExitIfErr(err)

			cfg.Bootstrap = bootstrap
		}

		if err := os.MkdirAll(QriRepoPath, os.ModePerm); err != nil {
			ErrExit(fmt.Errorf("error creating home dir: %s", err.Error()))
		}
		err = writeConfigFile(cfg)
		ExitIfErr(err)

		err = viper.ReadInConfig()
		ExitIfErr(err)

		if setupIdentityData != "" {
			err = readAtFile(&setupIdentityData)
			ExitIfErr(err)

			id := config.Identity{}
			err = json.Unmarshal([]byte(setupIdentityData), &id)
			ExitIfErr(err)

			path := filepath.Join(os.TempDir(), "config")
			data, err := json.Marshal(DefaultIPFSConfig(id))
			ExitIfErr(err)

			err = ioutil.WriteFile(path, data, os.ModePerm)
			ExitIfErr(err)

			setupIPFSConfigFile = path
			defer os.Remove(path)
		}

		if setupIPFS {
			err = ipfs.InitRepo(IpfsFsPath, setupIPFSConfigFile)
			ExitIfErr(err)
		}

		if setupProfileData != "" {
			err = readAtFile(&setupProfileData)
			ExitIfErr(err)

			p := &core.Profile{}
			err = json.Unmarshal([]byte(setupProfileData), p)
			ExitIfErr(err)

			pr, err := profileRequests(false)
			ExitIfErr(err)

			res := &core.Profile{}
			err = pr.SaveProfile(p, res)
			ExitIfErr(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(setupCmd)
	setupCmd.Flags().BoolVarP(&setupOverwrite, "overwrite", "", false, "overwrite repo if one exists")
	setupCmd.Flags().BoolVarP(&setupIPFS, "init-ipfs", "", true, "initialize an IPFS repo if one isn't present")
	// setupCmd.Flags().StringVarP(&setupIPFSConfigFile, "ipfs-config", "", "", "config file for setupialization")
	setupCmd.Flags().StringVarP(&setupIdentityData, "id", "", "", "json-encoded identity data, specify a filepath with '@' prefix")
	setupCmd.Flags().StringVarP(&setupProfileData, "profile", "", "", "json-encoded user profile data, specify a filepath with '@' prefix")
	setupCmd.Flags().StringVarP(&setupDatasetsData, "datasets", "", "", "json-encoded object of default datasets")
	setupCmd.Flags().StringVarP(&setupBootstrapData, "bootstrap", "", "", "json-encoded array of boostrap multiaddrs")
}

// QRIRepoInitialized checks to see if a repository has been initialized at $QRI_PATH
func QRIRepoInitialized() bool {
	// for now this just checks for an existing config file
	_, err := os.Stat(configFilepath())
	return !os.IsNotExist(err)
}

func mapEnvVars(vars map[string]*string) {
	for envVar, value := range vars {
		envVal := os.Getenv(envVar)
		if envVal != "" {
			fmt.Printf("reading %s from env\n", envVar)
			*value = envVal
		}
	}
}

func setupRepoIfEmpty(repoPath, configPath string) error {
	if repoPath != "" {
		if _, err := os.Stat(filepath.Join(repoPath, "config")); os.IsNotExist(err) {
			if err := os.MkdirAll(repoPath, os.ModePerm); err != nil {
				return err
			}
			if err := ipfs.InitRepo(repoPath, configPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// readAtFile is a unix curl inspired method. any data input that begins with "@"
// is assumed to instead be a filepath that should be read & replaced with the contents
// of the specified path
func readAtFile(data *string) error {
	d := *data
	if len(d) > 0 && d[0] == '@' {
		fileData, err := ioutil.ReadFile(d[1:])
		if err != nil {
			return err
		}
		*data = string(fileData)
	}
	return nil
}

// DefaultIPFSConfig returns the standard IPFS configuration
// TODO - this is a bit of a hack for the moment, will be removed later
// in favour of using IPFS Config package more directly.
func DefaultIPFSConfig(identity config.Identity) *config.Config {
	return &config.Config{
		Identity: identity,
		Datastore: config.Datastore{
			StorageMax:         "10GB",
			StorageGCWatermark: 90,
			GCPeriod:           "1h",
			Spec: map[string]interface{}{
				"mounts": []map[string]interface{}{
					{
						"child": map[string]interface{}{
							"path":      "blocks",
							"shardFunc": "/repo/flatfs/shard/v1/next-to-last/2",
							"sync":      true,
							"type":      "flatfs",
						},
						"mountpoint": "/blocks",
						"prefix":     "flatfs.datastore",
						"type":       "measure",
					},
					{
						"child": map[string]interface{}{
							"compression": "none",
							"path":        "datastore",
							"type":        "levelds",
						},
						"mountpoint": "/",
						"prefix":     "leveldb.datastore",
						"type":       "measure",
					},
				},
				"type": "mount",
			},
			HashOnRead:      false,
			BloomFilterSize: 0,
		},
		Bootstrap: []string{
			"/dnsaddr/bootstrap.libp2p.io/ipfs/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			"/dnsaddr/bootstrap.libp2p.io/ipfs/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
			"/dnsaddr/bootstrap.libp2p.io/ipfs/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
			"/dnsaddr/bootstrap.libp2p.io/ipfs/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
			"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
			"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
			"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
			"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
			"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
			"/ip6/2604:a880:1:20::203:d001/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
			"/ip6/2400:6180:0:d0::151:6001/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
			"/ip6/2604:a880:800:10::4a:5001/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
			"/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
		},
		// setup the node's default addresses.
		// NOTE: two swarm listen addrs, one tcp, one utp.
		Addresses: config.Addresses{
			Swarm: []string{
				"/ip4/0.0.0.0/tcp/4001",
				// "/ip4/0.0.0.0/udp/4002/utp", // disabled for now.
				"/ip6/::/tcp/4001",
			},
			Announce:   []string{},
			NoAnnounce: []string{},
			API:        "/ip4/127.0.0.1/tcp/5001",
			Gateway:    "/ip4/127.0.0.1/tcp/8080",
		},

		Discovery: config.Discovery{config.MDNS{
			Enabled:  true,
			Interval: 10,
		}},

		// setup the node mount points.
		Mounts: config.Mounts{
			IPFS: "/ipfs",
			IPNS: "/ipns",
		},

		Ipns: config.Ipns{
			ResolveCacheSize: 128,
		},

		Gateway: config.Gateway{
			RootRedirect: "",
			Writable:     false,
			PathPrefixes: []string{},
			HTTPHeaders: map[string][]string{
				"Access-Control-Allow-Origin":  []string{"*"},
				"Access-Control-Allow-Methods": []string{"GET"},
				"Access-Control-Allow-Headers": []string{"X-Requested-With", "Range"},
			},
		},
		Reprovider: config.Reprovider{
			Interval: "12h",
			Strategy: "all",
		},
		Swarm: config.SwarmConfig{
			ConnMgr: config.ConnMgr{
				LowWater:    config.DefaultConnMgrLowWater,
				HighWater:   config.DefaultConnMgrHighWater,
				GracePeriod: config.DefaultConnMgrGracePeriod.String(),
				Type:        "basic",
			},
		},
	}
}
