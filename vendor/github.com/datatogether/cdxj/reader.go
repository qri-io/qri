package cdxj

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// headerPrefix is the required prefix to be a valid cdxj file
var headerPrefix = []byte("!OpenWayback-CDXJ")

// A Reader reads records from a CDXJ-encoded io.Reader.
//
// As returned by NewReader, a Reader expects input conforming to RFC 4180.
// The exported fields can be changed to customize the details before the
// first call to Read or ReadAll.
type Reader struct {
	// record counter
	record int
	s      *bufio.Scanner
}

// NewReader returns a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		s: bufio.NewScanner(r),
	}
}

// Read reads a record from the reader
// err io.EOF will be returned when the last record is reached
func (r *Reader) Read() (*Record, error) {
	rec := &Record{}
	// scan until we have a non-header record
	for r.s.Scan() {
		if len(r.s.Bytes()) == 0 || bytes.HasPrefix(r.s.Bytes(), []byte("!")) {
			continue
		}
		break
	}
	if r.s.Err() != nil {
		return nil, r.s.Err()
	}

	if err := rec.UnmarshalCDXJ(r.s.Bytes()); err != nil {
		return nil, err
	}
	r.record++
	return rec, nil
}

// ReadAll consumes the entire reader, returning a slice of records
func (r *Reader) ReadAll() ([]*Record, error) {
	records := []*Record{}
	for {
		rec, err := r.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return records, err
		}

		records = append(records, rec)
	}

	return records, nil
}

// Validate checks that an io.Reader is a valid cdxj format
func Validate(r io.Reader) error {
	hasHeader := false
	s := bufio.NewScanner(r)
	// scan for header
	for s.Scan() {
		if s.Text() != "" {
			if !bytes.Contains([]byte(s.Text()), headerPrefix) {
				return fmt.Errorf("invalid format, missing cdxj header")
			}
			hasHeader = true
			break
		}
	}

	if !hasHeader {
		return fmt.Errorf("invalid format, missing cdxj header")
	}

	// TODO - validate rows
	return nil
}
