// cdx implements the CDXJ file format used by OpenWayback 3.0.0 (and later) to index web archive contents
// (notably in  WARC and ARC files) and make them searchable via a resource resolution service.
// The format builds on the CDX file format originally developed by the Internet Archive
// for the indexing behind the WaybackMachine.
// This specification builds on it by simplifying the primary fields while adding a flexible JSON 'block'
// to each record, allowing high flexiblity in the inclusion of additional data.
package cdxj

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/datatogether/warc"
	"github.com/puerkitobio/purell"
)

var CanonicalizationScheme = purell.FlagsSafe

// Following the header lines, each additional line should represent exactly one resource in a web archive.
// Typically in a WARC (ISO 28500) or ARC file, although the exact storage of the resource is not defined
// by this specification. Each such line shall be refered to as a *record*.
type Record struct {
	// Searchable URI
	// By *searchable*, we mean that the following transformations have been applied to it:
	// 1. Canonicalization - See Appendix A
	// 2. Sort-friendly URI Reordering Transform (SURT)
	// 3. The scheme is dropped from the SURT format
	Uri string
	// should correspond to the WARC-Date timestamp as of WARC 1.1.
	// The timestamp shall represent the instant that data capture for record
	// creation began.
	// All timestamps should be in UTC.
	Timestamp time.Time
	// Indicates what type of record the current line refers to.
	// This field is fully compatible with WARC 1.0 definition of
	// WARC-Type (chapter 5.5 and chapter 6).
	RecordType warc.RecordType
	// This should contain fully valid JSON data. The only limitation, beyond those
	// imposed by JSON encoding rules, is that this may not contain any newline
	// characters, either in Unix (0x0A) or Windows form (0x0D0A).
	// The first occurance of a 0x0A constitutes the end of this field (and the record).
	JSON map[string]interface{}
}

func CreateRecord(rec *warc.Record) (*Record, error) {
	can, err := CanonicalizeURL(rec.TargetUri())
	if err != nil {
		return nil, err
	}

	surt, err := SURTUrl(can)
	if err != nil {
		return nil, err
	}

	return &Record{
		Uri:        surt,
		Timestamp:  rec.Date(),
		RecordType: rec.Type,
		JSON:       map[string]interface{}{},
	}, nil
}

// UnmarshalCDXJ reads a cdxj record from a byte slice
func (r *Record) UnmarshalCDXJ(data []byte) (err error) {
	rdr := bytes.NewReader(data)
	buf := bufio.NewReader(rdr)

	surturl, err := buf.ReadString(' ')
	if err != nil {
		return err
	}
	r.Uri, err = UnSURTUrl(surturl)
	if err != nil {
		return err
	}

	ts, err := buf.ReadString(' ')
	if err != nil {
		return err
	}
	r.Timestamp, err = time.Parse(time.RFC3339, strings.TrimSpace(ts))
	if err != nil {
		return err
	}

	rt, err := buf.ReadString(' ')
	if err != nil {
		return err
	}
	r.RecordType = warc.ParseRecordType(strings.TrimSpace(rt))

	r.JSON = map[string]interface{}{}
	if err := json.NewDecoder(buf).Decode(&r.JSON); err != nil {
		return err
	}

	return nil
}

// MarshalCDXJ outputs a CDXJ representation of r
func (r *Record) MarshalCDXJ() ([]byte, error) {
	jb, err := json.Marshal(r.JSON)
	if err != nil {
		return nil, err
	}

	suri, err := SURTUrl(r.Uri)
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf("%s %s %s %s\n", suri, r.Timestamp.In(time.UTC).Format(time.RFC3339), r.RecordType, string(jb))), nil
}

// Canonicalization is applied to URIs to remove trivial
// differences in the URIs that do not  reflect that the
// URI reference different resources.
// Examples include removing session ID parameters,
// unneccessary port declerations (e.g. :80 when crawling HTTP).
// OpenWayback implements its own canonicalization process.
// Typically, it will be applied to the searchable URIs in CDXJ files. You can,
// however, use any canonicalization scheme you care for (including none).
// You must simply ensure that the same canonicalization process is
// applied to the URIs when performing searches.
// Otherwise they may not match correctly.
func CanonicalizeURL(rawurl string) (string, error) {
	return purell.NormalizeURLString(rawurl, CanonicalizationScheme)
}

// SURTUrl is a transformation applied to URIs which makes their left-to-right
// representation better match the natural hierarchy of domain  names.
// A URI `<scheme://domain.tld/path?query>` has SURT form `<scheme://(tld,domain,)/path?query>`.
// Conversion to SURT form also involves making all characters lowercase,
// and changing the 'https' scheme to 'http'. Further, the '/' after  a URI authority component --
// for example, the third slash in a regular HTTP URI -- will only appear in the SURT
// form if it appeared in the plain URI form.
func SURTUrl(rawurl string) (string, error) {
	rawurl = strings.ToLower(rawurl)

	// TODO - if the query param contains a url of some kind, and the scheme is missing
	// this will fail, probably going to need to use regex :/
	if !strings.Contains(rawurl, "://") {
		rawurl = "http://" + rawurl
	}

	u, err := url.Parse(rawurl)
	if err != nil {
		return rawurl, err
	}

	s := strings.Split(u.Hostname(), ".")
	reverseSlice(s)

	surt := fmt.Sprintf("(%s,)%s", strings.Join(s, ","), u.Path)
	if u.RawQuery != "" {
		surt += fmt.Sprintf("?%s", u.RawQuery)
	}

	surt += ">"

	return surt, nil
}

// UnSURTUrl turns a SURT'ed url back into a normal Url
// TODO - should accept SURT urls that contain a scheme
func UnSURTUrl(surturl string) (string, error) {
	surturl = strings.Trim(surturl, "(> \n")
	buf := strings.NewReader(surturl)
	s := bufio.NewReader(buf)

	base, err := s.ReadString(')')
	if err != nil {
		return surturl, err
	}
	sl := strings.Split(strings.Trim(base, ",)"), ",")
	reverseSlice(sl)
	hostname := strings.Join(sl, ".")

	return fmt.Sprintf("%s%s", hostname, surturl[len(base):]), nil
}

// UnSURTPath gives the path element of a SURT'ed url
func UnSURTPath(surturl string) (string, error) {
	surturl = strings.Trim(surturl, "(> \n")
	buf := strings.NewReader(surturl)
	s := bufio.NewReader(buf)

	base, err := s.ReadString(')')
	if err != nil {
		return surturl, err
	}

	path := surturl[len(base):]
	if len(path) == 0 || path[0] != '/' {
		path = "/" + path
	}

	return path, nil
}

// reverseSlice reverses a slice of strings
func reverseSlice(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
