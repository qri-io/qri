package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/viper"
)

// ErrExit writes an error to stdout & exits
func ErrExit(err error) {
	PrintErr(err)
	os.Exit(1)
}

func ExitIfErr(err error) {
	if err != nil {
		// PrintErr(err)
		panic(err)
		os.Exit(1)
	}
}

// GetWd is a convenience method to get the working
// directory or bail.
func GetWd() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %s", err.Error())
		os.Exit(1)
	}

	return dir
}

// func GetAddress(cmd *cobra.Command, args []string) dataset.Address {
// 	adr := dataset.NewAddress("")
// 	if len(args) > 0 {
// 		adr = dataset.NewAddress(args[0])
// 	}
// 	return adr
// }

// Store creates the appropriate store for a given command
// defaulting to creating a new store from the local directory
// func Store(cmd *cobra.Command, args []string) fs.Store {
// 	return local.NewLocalStore(cachePath())
// }

// Cache is the place to put downloaded stuff. default is the local store
// func Cache() fs.Store {
// 	return local.NewLocalStore(cachePath())
// }

// cachePath returns the configurable place to keep data
func cachePath() string {
	return viper.GetString("cache")
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

// func DatasetPath(ds *dataset.Dataset, elem ...string) string {
// 	return filepath.Join(append([]string{ds.Address.PathString()}, elem...)...)
// }

// func WriteDataset(store fs.Store, ds *dataset.Dataset, files map[string][]byte) error {
// 	if data, err := json.Marshal(ds); err != nil {
// 		return err
// 	} else {
// 		if err := store.Write(DatasetPath(ds, dataset.Filename), data); err != nil {
// 			return err
// 		}
// 	}

// 	for filename, data := range files {
// 		if err := store.Write(DatasetPath(ds, filename), data); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }
