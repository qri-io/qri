package memfs

import (
	"bytes"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"io/ioutil"
	"testing"
)

func TestMemFilestore(t *testing.T) {
	ms := NewMapstore()
	if err := RunFilestoreTests(ms); err != nil {
		t.Error(err.Error())
	}
}

func TestPathPrefix(t *testing.T) {
	got := NewMapstore().PathPrefix()
	if "map" != got {
		t.Errorf("path prefix mismatch. expected: 'map', got: %s", got)
	}
}

func RunFilestoreTests(f cafs.Filestore) error {
	if err := SingleFile(f); err != nil {
		return err
	}

	if err := Directory(f); err != nil {
		return err
	}

	if err := RunFilestoreAdderTests(f); err != nil {
		return err
	}

	return nil
}

func SingleFile(f cafs.Filestore) error {
	fdata := []byte("foo")
	file := NewMemfileBytes("file.txt", fdata)
	key, err := f.Put(file, false)
	if err != nil {
		return fmt.Errorf("Filestore.Put(%s) error: %s", file.FileName(), err.Error())
	}

	outf, err := f.Get(key)
	if err != nil {
		return fmt.Errorf("Filestore.Get(%s) error: %s", key.String(), err.Error())
	}
	data, err := ioutil.ReadAll(outf)
	if err != nil {
		return fmt.Errorf("error reading data from returned file: %s", err.Error())
	}
	if !bytes.Equal(fdata, data) {
		return fmt.Errorf("mismatched return value from get: %s != %s", string(fdata), string(data))
		// return fmt.Errorf("mismatched return value from get: %s != %s", outf.FileName(), string(data))
	}

	has, err := f.Has(datastore.NewKey("----------no-match---------"))
	if err != nil {
		return fmt.Errorf("Filestore.Has([nonexistent key]) error: %s", err.Error())
	}
	if has {
		return fmt.Errorf("filestore claims to have a very silly key")
	}

	has, err = f.Has(key)
	if err != nil {
		return fmt.Errorf("Filestore.Has(%s) error: %s", key.String(), err.Error())
	}
	if !has {
		return fmt.Errorf("Filestore.Has(%s) should have returned true", key.String())
	}
	if err = f.Delete(key); err != nil {
		return fmt.Errorf("Filestore.Delete(%s) error: %s", key.String(), err.Error())
	}

	return nil
}

func Directory(f cafs.Filestore) error {
	file := NewMemdir("/a",
		NewMemfileBytes("b.txt", []byte("a")),
		NewMemdir("c",
			NewMemfileBytes("d.txt", []byte("d")),
		),
		NewMemfileBytes("e.txt", []byte("e")),
	)
	key, err := f.Put(file, false)
	if err != nil {
		return fmt.Errorf("Filestore.Put(%s) error: %s", file.FileName(), err.Error())
	}

	outf, err := f.Get(key)
	if err != nil {
		return fmt.Errorf("Filestore.Get(%s) error: %s", key.String(), err.Error())
	}

	expectPaths := []string{
		"/a",
		"/a/b.txt",
		"/a/c",
		"/a/c/d.txt",
		"/a/e.txt",
	}

	paths := []string{}
	cafs.Walk(outf, 0, func(f cafs.File, depth int) error {
		paths = append(paths, f.FullPath())
		return nil
	})

	if len(paths) != len(expectPaths) {
		return fmt.Errorf("path length mismatch. expected: %d, got %d", len(expectPaths), len(paths))
	}

	for i, p := range expectPaths {
		if paths[i] != p {
			return fmt.Errorf("path %d mismatch expected: %s, got: %s", i, p, paths[i])
		}
	}

	if err = f.Delete(key); err != nil {
		return fmt.Errorf("Filestore.Delete(%s) error: %s", key.String(), err.Error())
	}

	return nil
}

func RunFilestoreAdderTests(f cafs.Filestore) error {
	adder, err := f.NewAdder(false, false)
	if err != nil {
		return fmt.Errorf("Filestore.NewAdder(false,false) error: %s", err.Error())
	}

	data := []byte("bar")
	if err := adder.AddFile(NewMemfileBytes("test.txt", data)); err != nil {
		return fmt.Errorf("Adder.AddFile error: %s", err.Error())
	}

	if err := adder.Close(); err != nil {
		return fmt.Errorf("Adder.Close() error: %s", err.Error())
	}

	return nil
}
