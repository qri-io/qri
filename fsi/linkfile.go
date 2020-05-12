package fsi

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/qri/dsref"
)

// RefLinkTextFilename is the filename for a reference linkfile
const RefLinkTextFilename = "qri-ref.txt"

// RefLinkHiddenFilename is the filename for a hidden reference linkfile
const RefLinkHiddenFilename = ".qri-ref"

// ReadLinkfile reads a reference from a linkfile with the given filename
func ReadLinkfile(filename string) (dsref.Ref, error) {
	var ref dsref.Ref
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return ref, err
	}
	return dsref.Parse(strings.TrimSpace(string(data)))
}

// LinkfileExistsInDir returns whether a linkfile exists in the directory
func LinkfileExistsInDir(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, RefLinkTextFilename)); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, RefLinkHiddenFilename)); err == nil {
		return true
	}
	return false
}

// WriteTextLinkfileInDir writes a reference to a visible linkfile in the given directory
func WriteTextLinkfileInDir(dir string, ref dsref.Ref) (string, error) {
	filename := filepath.Join(dir, RefLinkTextFilename)
	return filename, ioutil.WriteFile(filename, []byte(refText(ref)), 0644)
}

// WriteHiddenLinkfileInDir writes a reference to a hidden linkfile in the given directory
func WriteHiddenLinkfileInDir(dir string, ref dsref.Ref) (string, error) {
	filename := filepath.Join(dir, RefLinkHiddenFilename)
	return filename, WriteHiddenFile(filename, refText(ref))
}

// WriteRef writes a reference to the given io.Writer
func WriteRef(w io.Writer, ref dsref.Ref) {
	w.Write([]byte(refText(ref)))
}

func refText(ref dsref.Ref) string {
	if ref.Path != "" {
		return fmt.Sprintf("%s/%s@%s", ref.Username, ref.Name, ref.Path)
	}
	return fmt.Sprintf("%s/%s", ref.Username, ref.Name)
}
