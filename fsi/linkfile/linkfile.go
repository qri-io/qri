package linkfile

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/fsi/hiddenfile"
)

// RefLinkTextFilename is the filename for a reference linkfile
const RefLinkTextFilename = "qri-ref.txt"

// RefLinkHiddenFilename is the filename for a hidden reference linkfile
const RefLinkHiddenFilename = ".qri-ref"

// Read reads a reference from a linkfile with the given filename
func Read(filename string) (dsref.Ref, error) {
	var ref dsref.Ref
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return ref, err
	}
	return dsref.Parse(strings.TrimSpace(string(data)))
}

// ExistsInDir returns whether a linkfile exists in the directory
func ExistsInDir(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, RefLinkTextFilename)); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, RefLinkHiddenFilename)); err == nil {
		return true
	}
	return false
}

// WriteTextInDir writes a reference to a visible linkfile in the given directory
func WriteTextInDir(dir string, ref dsref.Ref) (string, error) {
	filename := filepath.Join(dir, RefLinkTextFilename)
	return filename, ioutil.WriteFile(filename, []byte(refText(ref)), 0644)
}

// WriteHiddenInDir writes a reference to a hidden linkfile in the given directory
func WriteHiddenInDir(dir string, ref dsref.Ref) (string, error) {
	filename := filepath.Join(dir, RefLinkHiddenFilename)
	return filename, hiddenfile.WriteHiddenFile(filename, refText(ref))
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
