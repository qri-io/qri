// cafs is a "content-addressed-file-systen", which is a generalized interface for
// working with content-addressed filestores.
// real-on-the-real, this is a wrapper for IPFS.
// It looks a lot like the ipfs datastore interface, except the datastore itself
// determines keys.
package cafs

import (
	"fmt"

	"github.com/ipfs/go-datastore"
)

var (
	ErrNotFound = fmt.Errorf("cafs: Not Found")
)

// Filestore is an interface for working with a content-addressed file system.
// This interface is under active development, expect it to change lots.
// It's currently form-fitting around IPFS (ipfs.io), with far-off plans to generalize
// toward compatibility with git (git-scm.com), then maybe other stuff, who knows.
type Filestore interface {
	// Put places a file or a directory in the store.
	// The most notable difference from a standard file store is the store itself determines
	// the resulting key (google "content addressing" for more info ;)
	// keys returned by put must be prefixed with the PathPrefix,
	// eg. /ipfs/QmZ3KfGaSrb3cnTriJbddCzG7hwQi2j6km7Xe7hVpnsW5S
	Put(file File, pin bool) (key datastore.Key, err error)

	// Get retrieves the object `value` named by `key`.
	// Get will return ErrNotFound if the key is not mapped to a value.
	Get(key datastore.Key) (file File, err error)

	// Has returns whether the `key` is mapped to a `value`.
	// In some contexts, it may be much cheaper only to check for existence of
	// a value, rather than retrieving the value itself. (e.g. HTTP HEAD).
	// The default implementation is found in `GetBackedHas`.
	Has(key datastore.Key) (exists bool, err error)

	// Delete removes the value for given `key`.
	Delete(key datastore.Key) error

	// NewAdder allocates an Adder instance for adding files to the filestore
	// Adder gives a higher degree of control over the file adding process at the
	// cost of being harder to work with.
	// "pin" is a flag for recursively pinning this object
	// "wrap" sets weather the top level should be wrapped in a directory
	// expect this to change to something like:
	// NewAdder(opt map[string]interface{}) (Adder, error)
	NewAdder(pin, wrap bool) (Adder, error)

	// PathPrefix is a top-level identifier to distinguish between filestores,
	// for exmple: the "ipfs" in /ipfs/QmZ3KfGaSrb3cnTriJbddCzG7hwQi2j6km7Xe7hVpnsW5S
	// a Filestore implementation should always return the same
	PathPrefix() string
}

// Fetcher is the interface for getting files from a remote source
// filestores can opt into the fetcher interface
type Fetcher interface {
	// Fetch gets a file from a source
	Fetch(source Source, key datastore.Key) (SizeFile, error)
}

// Source identifies where a file should come from.
// examples of different sources could be an HTTP url or P2P node Identifier
type Source interface {
	// address should return the base resource identifier in either content
	// or location based addressing schemes
	Address() string
}

// source is an internal implementation of the Source interface
type source string

func (s source) Address() string { return string(s) }

var (
	// SourceAny specifies that content can come from anywhere
	SourceAny = source("any")
)

// Pinner interface for content stores that support
// the concept of pinning (originated by IPFS).
type Pinner interface {
	Pin(key datastore.Key, recursive bool) error
	Unpin(key datastore.Key, recursive bool) error
}

// Adder is the interface for adding files to a Filestore. The addition process
// is parallelized. Implementers must make all required AddFile calls, then call
// Close to finalize the addition process. Progress can be monitored through the
// Added() channel
type Adder interface {
	// AddFile adds a file or directory of files to the store
	// this function will return immideately, consumers should read
	// from the Added() channel to see the results of file addition.
	AddFile(File) error
	// Added gives a channel to read added files from.
	Added() chan AddedFile
	// In IPFS land close calls adder.Finalize() and adder.PinRoot()
	// (files will only be pinned if the pin flag was set on NewAdder)
	// Close will close the underlying
	Close() error
}

// AddedFile reports on the results of adding a file to the store
// TODO - add filepath to this struct
type AddedFile struct {
	Path  datastore.Key
	Name  string
	Bytes int64
	Hash  string
	Size  string
}

// Walk traverses a file tree calling visit on each node
func Walk(root File, depth int, visit func(f File, depth int) error) (err error) {
	if err := visit(root, depth); err != nil {
		return err
	}

	if root.IsDirectory() {
		for {
			f, err := root.NextFile()
			if err != nil {
				if err.Error() == "EOF" {
					break
				} else {
					return err
				}
			}

			if err := Walk(f, depth+1, visit); err != nil {
				return err
			}
		}
	}
	return nil
}
