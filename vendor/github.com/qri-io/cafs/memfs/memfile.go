// memfs satsfies the ipfs cafs.File interface in memory
// An example pulled from tests will create a tree of "cafs"
// with directories & cafs, with paths properly set:
// NewMemdir("/a",
// 	NewMemfileBytes("a.txt", []byte("foo")),
// 	NewMemfileBytes("b.txt", []byte("bar")),
// 	NewMemdir("/c",
// 		NewMemfileBytes("d.txt", []byte("baz")),
// 		NewMemdir("/e",
// 			NewMemfileBytes("f.txt", []byte("bat")),
// 		),
// 	),
// )
// File is an interface that provides functionality for handling
// cafs/directories as values that can be supplied to commands.
//
// This is pretty close to things that already exist in ipfs
// and might not be necessary in most situations, but provides a sensible
// degree of modularity for our purposes:
// * memdir: github.com/ipfs/go-ipfs/commands/cafs.SerialFile
// * memfs: github.com/ipfs/go-ipfs/commands/cafs.ReaderFile
package memfs

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"

	"github.com/qri-io/cafs"
)

// PathSetter adds the capacity to modify a path property
type PathSetter interface {
	SetPath(path string)
}

// Memfile is an in-memory file
type Memfile struct {
	buf  io.Reader
	name string
	path string
}

// Confirm that Memfile satisfies the cafs.File interface
var _ = (cafs.File)(&Memfile{})

// NewMemfileBytes creates a file from an io.Reader
func NewMemfileReader(name string, r io.Reader) *Memfile {
	return &Memfile{
		buf:  r,
		name: name,
	}
}

// NewMemfileBytes creates a file from a byte slice
func NewMemfileBytes(name string, data []byte) *Memfile {
	return &Memfile{
		buf:  bytes.NewBuffer(data),
		name: name,
	}
}

func (m Memfile) Read(p []byte) (int, error) {
	return m.buf.Read(p)
}

func (m Memfile) Close() error {
	if closer, ok := m.buf.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (m Memfile) FileName() string {
	return m.name
}

func (m Memfile) FullPath() string {
	return m.path
}

func (m *Memfile) SetPath(path string) {
	m.path = path
}

func (Memfile) IsDirectory() bool {
	return false
}

func (Memfile) NextFile() (cafs.File, error) {
	return nil, cafs.ErrNotDirectory
}

// Memdir is an in-memory directory
// Currently it only supports either Memfile & Memdir as children
type Memdir struct {
	path     string
	fi       int // file index for reading
	children []cafs.File
}

// Confirm that Memdir satisfies the cafs.File interface
var _ = (cafs.File)(&Memdir{})

// NewMemdir creates a new Memdir, supplying zero or more children
func NewMemdir(path string, children ...cafs.File) *Memdir {
	m := &Memdir{
		path:     path,
		children: []cafs.File{},
	}
	m.AddChildren(children...)
	return m
}

func (Memdir) Close() error {
	return cafs.ErrNotReader
}

func (Memdir) Read([]byte) (int, error) {
	return 0, cafs.ErrNotReader
}

func (m Memdir) FileName() string {
	return filepath.Base(m.path)
}

func (m Memdir) FullPath() string {
	return m.path
}

func (Memdir) IsDirectory() bool {
	return true
}

func (d *Memdir) NextFile() (cafs.File, error) {
	if d.fi >= len(d.children) {
		d.fi = 0
		return nil, io.EOF
	}
	defer func() { d.fi++ }()
	return d.children[d.fi], nil
}

func (d *Memdir) SetPath(path string) {
	d.path = path
	for _, f := range d.children {
		if fps, ok := f.(PathSetter); ok {
			fps.SetPath(filepath.Join(path, f.FileName()))
		}
	}
}

// AddChildren allows any sort of file to be added, but only
// implementations that implement the PathSetter interface will have
// properly configured paths.
func (d *Memdir) AddChildren(fs ...cafs.File) {
	for _, f := range fs {
		if fps, ok := f.(PathSetter); ok {
			fps.SetPath(filepath.Join(d.FullPath(), f.FileName()))
		}
		dir := d.MakeDirP(f)
		dir.children = append(dir.children, f)
	}
}

func (d *Memdir) ChildDir(dirname string) *Memdir {
	if dirname == "" || dirname == "." || dirname == "/" {
		return d
	}
	for _, f := range d.children {
		if dir, ok := f.(*Memdir); ok {
			if filepath.Base(dir.path) == dirname {
				return dir
			}
		}
	}
	return nil
}

func (d *Memdir) MakeDirP(f cafs.File) *Memdir {
	dirpath, _ := filepath.Split(f.FileName())
	if dirpath == "" {
		return d
	}
	dirs := strings.Split(dirpath[1:len(dirpath)-1], "/")
	if len(dirs) == 1 {
		return d
	}

	dir := d
	for _, dirname := range dirs {
		if ch := dir.ChildDir(dirname); ch != nil {
			dir = ch
			continue
		}
		ch := NewMemdir(filepath.Join(dir.FullPath(), dirname))
		dir.children = append(dir.children, ch)
		dir = ch
	}
	return dir
}
