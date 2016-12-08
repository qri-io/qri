package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/qri-io/dataset"
	"github.com/qri-io/fs"
	"github.com/qri-io/fs/local"
	"github.com/qri-io/namespace"
	lns "github.com/qri-io/namespace/local"
	"github.com/qri-io/namespace/remote"
	"github.com/spf13/cobra"
)

// ErrExit writes an error to stdout & exits
func ErrExit(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
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

// Store creates the appropriate store for a given command
// defaulting to creating a new store from the local directory
func Store(cmd *cobra.Command, args []string) fs.Store {
	return local.NewLocalStore(GetWd())
}

// Cache is the place to put downloaded stuff. default is the local store
func Cache() fs.Store {
	return local.NewLocalStore(GetWd())
}

// Namespaces is a collection of namespaces that also satisfies the namespace interface
// by querying each namespace in order
type Namespaces []namespace.Namespace

func (n Namespaces) Url() string {
	str := ""
	for _, ns := range n {
		str += ns.Url() + "\n"
	}
	return str
}
func (n Namespaces) Base() dataset.Address {
	// str := ""
	// for _, ns := range n {
	// 	str += ns.Base().String() + "\n"
	// }
	return dataset.NewAddress("")
}
func (n Namespaces) String() string {
	str := ""
	for _, ns := range n {
		str += ns.String() + "\n"
	}
	return str
}

func (n Namespaces) ChildAddresses(adr dataset.Address) (namespace.Addresses, error) {
	for _, ns := range n {
		if ds, err := ns.ChildAddresses(adr); err == nil {
			return ds, nil
		}
	}
	return nil, namespace.ErrNotFound
}

func (n Namespaces) ChildDatasets(adr dataset.Address) (namespace.Datasets, error) {
	for _, ns := range n {
		if ds, err := ns.ChildDatasets(adr); err == nil {
			return ds, nil
		}
	}
	return nil, namespace.ErrNotFound
}

func (n Namespaces) Dataset(adr dataset.Address) (*dataset.Dataset, error) {
	for _, ns := range n {
		if ds, err := ns.Dataset(adr); err == nil {
			return ds, nil
		}
	}
	return nil, namespace.ErrNotFound
}

func (n Namespaces) Package(adr dataset.Address) (io.ReaderAt, int64, error) {
	for _, ns := range n {
		if ds, size, err := ns.Package(adr); err == nil {
			return ds, size, nil
		}
	}

	return nil, 0, namespace.ErrNotFound
}

func (n Namespaces) Store(adr dataset.Address) (fs.Store, error) {
	for _, ns := range n {
		if _, err := ns.Dataset(adr); err == nil {
			// if the base is local, we can just hand back the local store
			if lcl, ok := ns.(*lns.Namespace); ok {
				return lcl.Store(adr)
			}

			// otherwise we need to download the dataset to our local store
			store, err := downloadPackage(ns, adr)
			if err != nil {
				return nil, err
			}
			return store, nil
		}
	}

	return nil, namespace.ErrNotFound
}

// Namespaces reads the list of namespaces from the config
func GetNamespaces(cmd *cobra.Command, args []string) Namespaces {
	return Namespaces{
		remote.New("localhost", "qri"),
		// lns.NewNamespaceFromPath(GetWd()),
	}
}

func downloadPackage(ns namespace.Namespace, adr dataset.Address) (fs.Store, error) {
	store := Cache()
	r, size, err := ns.Package(adr)
	if err != nil {
		return store, err
	}

	buf := make([]byte, size)
	if bytesRead, err := r.ReadAt(buf, 0); err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("read %d bytes\n", bytesRead)
	}

	if err := ioutil.WriteFile("test.zip", buf, os.ModePerm); err != nil {
		fmt.Println(err)
	}

	zipr, err := zip.NewReader(r, size)
	if err != nil {
		return store, err
	}

	for _, f := range zipr.File {
		r, err := f.Open()
		if err != nil {
			return store, err
		}

		data, err := ioutil.ReadAll(r)
		if err != nil {
			return store, err
		}

		if err := store.Write(f.Name, data); err != nil {
			return store, err
		}
	}

	return store, nil
}
