package startf

import (
	"fmt"
	"io"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/starlib/util"
	"go.starlark.net/starlark"
)

// EntryReader implements the dsio.EntryReader interface for starlark.Iterable's
type EntryReader struct {
	i    int
	st   *dataset.Structure
	iter starlark.Iterator
	data starlark.Value
}

// NewEntryReader creates a new Entry Reader
func NewEntryReader(st *dataset.Structure, iter starlark.Iterable) *EntryReader {
	return &EntryReader{
		st:   st,
		data: iter.(starlark.Value),
		iter: iter.Iterate(),
	}
}

// Structure gives this reader's structure
func (r *EntryReader) Structure() *dataset.Structure {
	return r.st
}

// ReadEntry reads one entry from the reader
func (r *EntryReader) ReadEntry() (e dsio.Entry, err error) {
	// Read next element (key for object, value for array).
	var next starlark.Value
	if !r.iter.Next(&next) {
		r.iter.Done()
		return e, io.EOF
	}

	// Handle array entry.
	tlt, err := dsio.GetTopLevelType(r.st)
	if err != nil {
		return
	}
	if tlt == "array" {
		e.Index = r.i
		r.i++
		e.Value, err = util.Unmarshal(next)
		if err != nil {
			fmt.Printf("reading error: %s\n", err.Error())
		}
		return
	}

	// Handle object entry. Assume key is a string.
	var ok bool
	e.Key, ok = starlark.AsString(next)
	if !ok {
		fmt.Printf("key error: %s\n", next)
	}
	// Lookup the corresponding value for the key.
	dict := r.data.(*starlark.Dict)
	value, ok, err := dict.Get(next)
	if err != nil {
		fmt.Printf("reading error: %s\n", err.Error())
	}
	e.Value, err = util.Unmarshal(value)
	if err != nil {
		fmt.Printf("reading error: %s\n", err.Error())
	}
	return
}

// Close finalizes the reader
func (r *EntryReader) Close() error {
	// TODO (b5): consume & close iterator
	return nil
}
