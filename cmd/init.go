package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/qri/core"
	"io/ioutil"
	"os"
	"path/filepath"

	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	config "gx/ipfs/QmViBzgruNUoLNBnXcx8YWbDNwV8MNGEGKkLo6JGetygdw/go-ipfs/repo/config"
)

var (
	initOverwrite      bool
	initIPFS           bool
	initIPFSConfigFile string
	initIdentityData   string
	initProfileData    string
	initDatasetsData   string
	initBootstrapData  string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a qri repo",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var cfgData []byte

		if QRIRepoInitialized() && !initOverwrite {
			// use --overwrite to overwrite this repo, erasing all data and deleting your account for good
			ErrExit(fmt.Errorf("repo already initialized."))
		}
		fmt.Println("initializing qri repo")

		envVars := map[string]*string{
			"QRI_INIT_IDENTITY_DATA":  &initIdentityData,
			"QRI_INIT_PROFILE_DATA":   &initProfileData,
			"QRI_INIT_DATASETS_DATA":  &initDatasetsData,
			"QRI_INIT_BOOTSTRAP_DATA": &initBootstrapData,
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

		if initDatasetsData != "" {
			err = readAtFile(&initDatasetsData)
			ExitIfErr(err)

			datasets := map[string]string{}
			err = json.Unmarshal([]byte(initDatasetsData), &datasets)
			ExitIfErr(err)

			cfg.DefaultDatasets = datasets
		}

		if initBootstrapData != "" {
			err = readAtFile(&initBootstrapData)
			ExitIfErr(err)

			boostrap := []string{}
			err = json.Unmarshal([]byte(initBootstrapData), &boostrap)
			ExitIfErr(err)

			cfg.Bootstrap = boostrap
		}

		if err := os.MkdirAll(QriRepoPath, os.ModePerm); err != nil {
			ErrExit(fmt.Errorf("error creating home dir: %s\n", err.Error()))
		}
		err = WriteConfigFile(cfg)
		ExitIfErr(err)

		err = viper.ReadInConfig()
		ExitIfErr(err)

		if initIdentityData != "" {
			err = readAtFile(&initIdentityData)
			ExitIfErr(err)

			id := config.Identity{}
			err = json.Unmarshal([]byte(initIdentityData), &id)
			ExitIfErr(err)

			path := filepath.Join(os.TempDir(), "config")
			data, err := json.Marshal(DefaultIPFSConfig(id))
			ExitIfErr(err)

			err = ioutil.WriteFile(path, data, os.ModePerm)
			ExitIfErr(err)

			initIPFSConfigFile = path
			defer os.Remove(path)
		}

		if initIPFS {
			err = ipfs.InitRepo(IpfsFsPath, initIPFSConfigFile)
			ExitIfErr(err)
		}

		if initProfileData != "" {
			err = readAtFile(&initProfileData)
			ExitIfErr(err)

			p := &core.Profile{}
			err = json.Unmarshal([]byte(initProfileData), p)
			ExitIfErr(err)

			pr, err := ProfileRequests(false)
			ExitIfErr(err)

			res := &core.Profile{}
			err = pr.SaveProfile(p, res)
			ExitIfErr(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&initOverwrite, "overwrite", "", false, "overwrite repo if one exists")
	initCmd.Flags().BoolVarP(&initIPFS, "init-ipfs", "", true, "initialize an IPFS repo if one isn't present")
	// initCmd.Flags().StringVarP(&initIPFSConfigFile, "ipfs-config", "", "", "config file for initialization")
	initCmd.Flags().StringVarP(&initIdentityData, "id", "", "", "json-encoded identity data, specify a filepath with '@' prefix")
	initCmd.Flags().StringVarP(&initProfileData, "profile", "", "", "json-encoded user profile data, specify a filepath with '@' prefix")
	initCmd.Flags().StringVarP(&initDatasetsData, "datasets", "", "", "json-encoded object of default datasets")
	initCmd.Flags().StringVarP(&initBootstrapData, "bootstrap", "", "", "json-encoded array of boostrap multiaddrs")
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

func initRepoIfEmpty(repoPath, configPath string) error {
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

// TODO - this is a bit of a hack for the moment, will be removed later
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
