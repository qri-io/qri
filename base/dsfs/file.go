package dsfs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"time"

	"github.com/qri-io/qfs"
)

// JSONFile is a convenenience method for creating a file from a json.Marshaller
func JSONFile(name string, m json.Marshaler) (fs.File, error) {
	data, err := m.MarshalJSON()
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}
	return NewMemfileBytes(name, data), nil
}

func fileBytes(file qfs.File, err error) ([]byte, error) {
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}
	return ioutil.ReadAll(file)
}

type fsFileInfo struct {
	name  string      // base name of the file
	size  int64       // length in bytes for regular files; system-dependent for others
	mode  fs.FileMode // file mode bits
	mtime time.Time   // modification time
	sys   interface{}
}

var _ os.FileInfo = (*fsFileInfo)(nil)

func (fi fsFileInfo) Name() string       { return fi.name }
func (fi fsFileInfo) Size() int64        { return fi.size }
func (fi fsFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi fsFileInfo) ModTime() time.Time { return fi.mtime }
func (fi fsFileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi fsFileInfo) Sys() interface{}   { return fi.sys }

func (fi *fsFileInfo) SetFilename(name string) error {
	fi.name = name
	return nil
}

type fsDirEntry struct {
	name   string
	isFile bool
}

var _ fs.DirEntry = (*fsDirEntry)(nil)

func (de fsDirEntry) Name() string { return de.name }
func (de fsDirEntry) IsDir() bool  { return !de.isFile }
func (ds fsDirEntry) Type() fs.FileMode {
	if ds.isFile {
		return 0
	}
	return fs.ModeDir
}
func (ds fsDirEntry) Info() (fs.FileInfo, error) { return nil, errors.New("fsDirEntry.FileInfo") }

// memfile is an in-memory file
type memfile struct {
	fi  os.FileInfo
	buf io.Reader
}

// Confirm that memfile satisfies the File interface
var _ = (fs.File)(&memfile{})

// NewFileWithInfo creates a new open file with provided file information
func NewFileWithInfo(fi fs.FileInfo, r io.Reader) (fs.File, error) {
	switch fi.Mode() {
	case os.ModeDir:
		return nil, fmt.Errorf("NewFileWithInfo doesn't support creating directories")
	default:
		return &memfile{
			fi:  fi,
			buf: r,
		}, nil
	}
}

// NewMemfileReader creates a file from an io.Reader
func NewMemfileReader(name string, r io.Reader) fs.File {
	return &memfile{
		fi: &fsFileInfo{
			name: name,
			size: int64(-1),
			mode: 0,
			// mtime: Timestamp(),
		},
		buf: r,
	}
}

// NewMemfileBytes creates a file from a byte slice
func NewMemfileBytes(name string, data []byte) fs.File {
	return &memfile{
		fi: &fsFileInfo{
			name: name,
			size: int64(len(data)),
			mode: 0,
			// mtime: Timestamp(),
		},
		buf: bytes.NewBuffer(data),
	}
}

// Stat returns information for this file
func (m memfile) Stat() (fs.FileInfo, error) {
	return m.fi, nil
}

// Read implements the io.Reader interface
func (m memfile) Read(p []byte) (int, error) {
	return m.buf.Read(p)
}

// Close closes the file, if the backing reader implements the io.Closer interface
// it will call close on the backing Reader
func (m memfile) Close() error {
	if closer, ok := m.buf.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
