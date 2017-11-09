package memfs

import (
	"github.com/qri-io/cafs"
	"testing"
)

func TestMemfile(t *testing.T) {
	a := NewMemdir("/a",
		NewMemfileBytes("a.txt", []byte("foo")),
		NewMemfileBytes("b.txt", []byte("bar")),
		NewMemdir("/c",
			NewMemfileBytes("d.txt", []byte("baz")),
			NewMemdir("/e",
				NewMemfileBytes("f.txt", []byte("bat")),
			),
		),
		NewMemfileBytes("h.txt", []byte("bong")),
		NewMemfileBytes("/i/j.txt", []byte("boink")),
	)

	a.AddChildren(NewMemfileBytes("g.txt", []byte("kazam")))

	expectPaths := []string{
		"/a",
		"/a/a.txt",
		"/a/b.txt",
		"/a/c",
		"/a/c/d.txt",
		"/a/c/e",
		"/a/c/e/f.txt",
		"/a/h.txt",
		"/a/i/j.txt",
		"/a/g.txt",
	}

	paths := []string{}
	err := cafs.Walk(a, 0, func(f cafs.File, depth int) error {
		paths = append(paths, f.FullPath())
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if len(paths) != len(expectPaths) {
		t.Errorf("path length mismatch. expected: %d, got %d", len(expectPaths), len(paths))
		return
	}

	for i, p := range expectPaths {
		if paths[i] != p {
			t.Errorf("path %d mismatch expected: %s, got: %s", i, p, paths[i])
		}
	}
}

func TestMemdirMakeDirP(t *testing.T) {
	dir := NewMemdir("/")
	dir.MakeDirP(NewMemfileBytes("./a/b/c/d/file.txt", []byte("foo")))
	dir.MakeDirP(NewMemfileBytes("./a/b/file.txt", []byte("foo")))

	expectPaths := []string{
		"/",
		"/a",
		"/a/b",
		"/a/b/c",
		"/a/b/c/d",
		// "/a/b/c/d/file.txt",
	}

	paths := []string{}
	err := cafs.Walk(dir, 0, func(f cafs.File, depth int) error {
		paths = append(paths, f.FullPath())
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if len(paths) != len(expectPaths) {
		t.Errorf("path length mismatch. expected: %d, got %d", len(expectPaths), len(paths))
		return
	}

	for i, p := range expectPaths {
		if paths[i] != p {
			t.Errorf("path %d mismatch expected: %s, got: %s", i, p, paths[i])
		}
	}
}
