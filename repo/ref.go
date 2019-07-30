package repo

import (
	"fmt"
	"regexp"
	"strings"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/mr-tron/base58/base58"
	"github.com/multiformats/go-multihash"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo/profile"
	repofb "github.com/qri-io/qri/repo/repo_fbs"
)

// Refstore keeps a collection of dataset references, Refstores require complete
// references (with both alias and identifiers), and can carry only one of a
// given alias eg: putting peer/dataset@a/ipfs/b when a ref with alias peer/dataset
// is already in the store will overwrite the stored reference
type Refstore interface {
	// PutRef adds a reference to the store. References must be complete with
	// Peername, Name, and Path specified
	PutRef(ref DatasetRef) error
	// GetRef "completes" a passed in alias (DatasetRef with at least Peername
	// and Name field specified), filling in missing fields with a stored ref
	// TODO - should we rename this to "CompleteRef"?
	GetRef(ref DatasetRef) (DatasetRef, error)
	// DeleteRef removes a reference from the store
	DeleteRef(ref DatasetRef) error
	// References returns a set of references from the store
	References(offset, limit int) ([]DatasetRef, error)
	// RefCount returns the number of references in the store
	RefCount() (int, error)
}

var isRefString = regexp.MustCompile(`^((\w+)\/(\w+)){0,1}(@(\w*)(\/\w{0,4}\/\w+)){0,1}$`)

// IsRefString checks to see if a reference is a valid dataset ref string
func IsRefString(path string) bool {
	return isRefString.MatchString(path)
}

// DatasetRef encapsulates a reference to a dataset. This needs to exist to bind
// ways of referring to a dataset to a dataset itself, as datasets can't easily
// contain their own hash information, and names are unique on a per-repository
// basis.
// It's tempting to think this needs to be "bigger", supporting more fields,
// keep in mind that if the information is important at all, it should
// be stored as metadata within the dataset itself.
type DatasetRef struct {
	// Peername of dataset owner
	Peername string `json:"peername,omitempty"`
	// ProfileID of dataset owner
	ProfileID profile.ID `json:"profileID,omitempty"`
	// Unique name reference for this dataset
	Name string `json:"name,omitempty"`
	// Content-addressed path for this dataset
	Path string `json:"path,omitempty"`
	// FSIPath is this dataset's link to the local filesystem if one exists
	FSIPath string `json:"fsiPath,omitempty"`
	// Dataset is a pointer to the dataset being referenced
	Dataset *dataset.Dataset `json:"dataset,omitempty"`
	// Published indicates whether this reference is listed as an available dataset
	Published bool `json:"published"`
}

// String implements the Stringer interface for DatasetRef
func (r DatasetRef) String() (s string) {
	s = r.AliasString()
	if r.ProfileID.String() != "" || r.Path != "" {
		s += "@"
	}
	if r.ProfileID.String() != "" {
		s += r.ProfileID.String()
	}
	if r.Path != "" {
		s += r.Path
	}
	return
}

// AliasString returns the alias components of a DatasetRef as a string
func (r DatasetRef) AliasString() (s string) {
	s = r.Peername
	if r.Name != "" {
		s += "/" + r.Name
	}
	return
}

// Complete returns true if a dataset has Peername, Name, ProfileID and Path
// properties set
func (r DatasetRef) Complete() bool {
	return r.Peername != "" && r.ProfileID != "" && r.Name != "" && r.Path != ""
}

// Match checks returns true if Peername and Name are equal,
// and/or path is equal
func (r DatasetRef) Match(b DatasetRef) bool {
	// fmt.Printf("\nr.Peername: %s b.Peername: %s\n", r.Peername, b.Peername)
	// fmt.Printf("\nr.Name: %s b.Name: %s\n", r.Name, b.Name)
	return (r.Path != "" && b.Path != "" && r.Path == b.Path) || (r.ProfileID == b.ProfileID || r.Peername == b.Peername) && r.Name == b.Name
}

// Equal returns true only if Peername Name and Path are equal
func (r DatasetRef) Equal(b DatasetRef) bool {
	return r.Peername == b.Peername && r.ProfileID == b.ProfileID && r.Name == b.Name && r.Path == b.Path
}

// IsPeerRef returns true if only Peername is set
func (r DatasetRef) IsPeerRef() bool {
	return (r.Peername != "" || r.ProfileID != "") && r.Name == "" && r.Path == "" && r.Dataset == nil
}

// IsEmpty returns true if none of it's fields are set
func (r DatasetRef) IsEmpty() bool {
	return r.Equal(DatasetRef{})
}

// MustParseDatasetRef panics if the reference is invalid. Useful for testing
func MustParseDatasetRef(refstr string) DatasetRef {
	ref, err := ParseDatasetRef(refstr)
	if err != nil {
		panic(err)
	}
	return ref
}

// ParseDatasetRef decodes a dataset reference from a string value
// It’s possible to refer to a dataset in a number of ways.
// The full definition of a dataset reference is as follows:
//     dataset_reference = peer_name/dataset_name@peer_id/network/hash
//
// we swap in defaults as follows, all of which are represented as
// empty strings:
//     network - defaults to /ipfs/
//     hash - tip of version history (latest known commit)
//
// these defaults are currently enforced by convention.
// TODO - make Dataset Ref parsing the responisiblity of the Repo
// interface, replacing empty strings with actual defaults
//
// dataset names & hashes are disambiguated by checking if the input
// parses to a valid multihash after base58 decoding.
// through defaults & base58 checking the following should all parse:
//     peer_name/dataset_name
//     /network/hash
//     peername
//     peer_id
//     @peer_id
//     @peer_id/network/hash
//
// see tests for more exmples
//
// TODO - add validation that prevents peernames from being
// valid base58 multihashes and makes sure hashes are actually valid base58 multihashes
// TODO - figure out how IPFS CID's play into this
func ParseDatasetRef(ref string) (DatasetRef, error) {
	if ref == "" {
		return DatasetRef{}, ErrEmptyRef
	}

	var (
		// nameRefString string
		dsr = DatasetRef{}
		err error
	)

	// if there is an @ symbol, we are dealing with a DatasetRef
	// with an identifier
	atIndex := strings.Index(ref, "@")

	if atIndex != -1 {

		dsr.Peername, dsr.Name = parseAlias(ref[:atIndex])
		dsr.ProfileID, dsr.Path, err = parseIdentifiers(ref[atIndex+1:])

	} else {

		var peername, datasetname, pid bool
		toks := strings.Split(ref, "/")

		for i, tok := range toks {
			if isBase58Multihash(tok) {
				// first hash we encounter is a peerID
				if !pid {
					dsr.ProfileID, _ = profile.IDB58Decode(tok)
					pid = true
					continue
				}

				if !isBase58Multihash(toks[i-1]) {
					dsr.Path = fmt.Sprintf("/%s/%s", toks[i-1], strings.Join(toks[i:], "/"))
				} else {
					dsr.Path = fmt.Sprintf("/ipfs/%s", strings.Join(toks[i:], "/"))
				}
				break
			}

			if !peername {
				dsr.Peername = tok
				peername = true
				continue
			}

			if !datasetname {
				dsr.Name = tok
				datasetname = true
				continue
			}

			dsr.Path = strings.Join(toks[i:], "/")
			break
		}
	}

	if dsr.ProfileID == "" && dsr.Peername == "" && dsr.Name == "" && dsr.Path == "" {
		err = fmt.Errorf("malformed DatasetRef string: %s", ref)
		return dsr, err
	}

	// if dsr.ProfileID != "" {
	// 	if !isBase58Multihash(dsr.ProfileID) {
	// 		err = fmt.Errorf("invalid ProfileID: '%s'", dsr.ProfileID)
	// 		return dsr, err
	// 	}
	// }

	return dsr, err
}

func parseAlias(alias string) (peer, dataset string) {
	for i, tok := range strings.Split(alias, "/") {
		switch i {
		case 0:
			peer = tok
		case 1:
			dataset = tok
		}
	}
	return
}

func parseIdentifiers(ids string) (profileID profile.ID, path string, err error) {

	toks := strings.Split(ids, "/")
	switch len(toks) {
	case 0:
		err = fmt.Errorf("malformed DatasetRef identifier: %s", ids)
	case 1:
		if toks[0] != "" {
			profileID, err = profile.IDB58Decode(toks[0])
			// if !isBase58Multihash(toks[0]) {
			// 	err = fmt.Errorf("'%s' is not a base58 multihash", ids)
			// }

			return
		}
	case 2:
		if pid, e := profile.IDB58Decode(toks[0]); e == nil {
			profileID = pid
		}

		if isBase58Multihash(toks[0]) && isBase58Multihash(toks[1]) {
			toks[1] = fmt.Sprintf("/ipfs/%s", toks[1])
		}

		path = toks[1]
	default:
		if pid, e := profile.IDB58Decode(toks[0]); e == nil {
			profileID = pid
		}

		path = fmt.Sprintf("/%s/%s", toks[1], toks[2])
		return
	}

	return
}

// TODO - this could be more robust?
func stripProtocol(ref string) string {
	if strings.HasPrefix(ref, "/ipfs/") {
		return ref[len("/ipfs/"):]
	}
	return ref
}

func isBase58Multihash(hash string) bool {
	data, err := base58.Decode(hash)
	if err != nil {
		return false
	}
	if _, err := multihash.Decode(data); err != nil {
		return false
	}

	return true
}

// CanonicalizeDatasetRef uses the user's repo to turn any local aliases into full dataset
// references using known canonical peernames and paths. If the provided reference is not
// in the local repo, still do the work of handling aliases, but return a repo.ErrNotFound
// error, which callers can respond to by possibly contacting remote repos.
func CanonicalizeDatasetRef(r Repo, ref *DatasetRef) error {
	if ref.IsEmpty() {
		return ErrEmptyRef
	}

	if err := CanonicalizeProfile(r, ref); err != nil {
		return err
	}

	if ref.Path != "" && ref.ProfileID != "" && ref.Name != "" && ref.Peername != "" {
		return nil
	}

	got, err := r.GetRef(*ref)
	if err != nil {
		return err
	}

	// TODO (b5) - this is the assign pattern, refactor into a method on DatasetRef
	if ref.Path == "" {
		ref.Path = got.Path
	}
	if ref.ProfileID == "" {
		ref.ProfileID = got.ProfileID
	}
	if ref.Name == "" {
		ref.Name = got.Name
	}
	if ref.Peername == "" || ref.Peername != got.Peername {
		ref.Peername = got.Peername
	}
	ref.Published = got.Published
	if ref.FSIPath == "" {
		ref.FSIPath = got.FSIPath
	}
	if ref.Path != got.Path || ref.ProfileID != got.ProfileID || ref.Name != got.Name {
		return fmt.Errorf("Given datasetRef %s does not match datasetRef on file: %s", ref.String(), got.String())
	}

	return nil
}

// CanonicalizeProfile populates dataset DatasetRef ProfileID and Peername properties,
// changing aliases to known names, and adding ProfileID from a peerstore
func CanonicalizeProfile(r Repo, ref *DatasetRef) error {
	if ref.Peername == "" && ref.ProfileID == "" {
		return nil
	}

	p, err := r.Profile()
	if err != nil {
		return err
	}

	// If this is a dataset ref that a peer of the user owns.
	if ref.Peername == "me" || ref.Peername == p.Peername || ref.ProfileID == p.ID {
		if ref.Peername == "me" {
			ref.ProfileID = p.ID
			ref.Peername = p.Peername
		}

		if ref.Peername != "" && ref.ProfileID != "" {
			if ref.Peername == p.Peername && ref.ProfileID != p.ID {
				return fmt.Errorf("Peername and ProfileID combination not valid: Peername = %s, ProfileID = %s, but was given ProfileID = %s", p.Peername, p.ID, ref.ProfileID)
			}
			if ref.ProfileID == p.ID && ref.Peername != p.Peername {
				// The peer renamed itself at some point, but the profileID matches. Use the
				// new peername.
				ref.Peername = p.Peername
				return nil
			}
			if ref.Peername == p.Peername && ref.ProfileID == p.ID {
				return nil
			}
		}

		if ref.Peername != "" {
			if ref.Peername != p.Peername {
				return nil
			}
		}

		if ref.ProfileID != "" {
			if ref.ProfileID != p.ID {
				return nil
			}
		}

		ref.Peername = p.Peername
		ref.ProfileID = p.ID
		return nil
	}

	if ref.ProfileID != "" {
		if profile, err := r.Profiles().GetProfile(ref.ProfileID); err == nil {

			if ref.Peername == "" {
				ref.Peername = profile.Peername
				return nil
			}
			if ref.Peername != profile.Peername {
				return fmt.Errorf("Peername and ProfileID combination not valid: ProfileID = %s, Peername = %s, but was given Peername = %s", profile.ID, profile.Peername, ref.Peername)
			}
		}
	}

	if ref.Peername != "" {
		if id, err := r.Profiles().PeernameID(ref.Peername); err == nil {
			// if err != nil {
			// 	return fmt.Errorf("error fetching peer from store: %s", err)
			// }
			if ref.ProfileID == "" {
				ref.ProfileID = id
				return nil
			}
			if ref.ProfileID != id {
				return fmt.Errorf("Peername and ProfileID combination not valid: Peername = %s, ProfileID = %s, but was given ProfileID = %s", ref.Peername, id.String(), ref.ProfileID)
			}
		}
	}
	return nil
}

// CompareDatasetRef compares two Dataset References, returning an error
// describing any difference between the two references
func CompareDatasetRef(a, b DatasetRef) error {
	if a.ProfileID != b.ProfileID {
		return fmt.Errorf("PeerID mismatch. %s != %s", a.ProfileID, b.ProfileID)
	}
	if a.Peername != b.Peername {
		return fmt.Errorf("Peername mismatch. %s != %s", a.Peername, b.Peername)
	}
	if a.Name != b.Name {
		return fmt.Errorf("Name mismatch. %s != %s", a.Name, b.Name)
	}
	if a.Path != b.Path {
		return fmt.Errorf("Path mismatch. %s != %s", a.Path, b.Path)
	}
	if a.Published != b.Published {
		return fmt.Errorf("Published mismatch: %t != %t", a.Published, b.Published)
	}
	return nil
}

// FlatbufferBytes formats a ref as a flatbuffer byte slice
func (r DatasetRef) FlatbufferBytes() []byte {
	builder := flatbuffers.NewBuilder(0)
	off := r.MarshalFlatbuffer(builder)
	builder.Finish(off)
	return builder.FinishedBytes()
}

// MarshalFlatbuffer writes a ref to a builder
func (r DatasetRef) MarshalFlatbuffer(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	peername := builder.CreateString(r.Peername)
	profileID := builder.CreateString(r.ProfileID.String())
	name := builder.CreateString(r.Name)
	path := builder.CreateString(r.Path)
	fsiPath := builder.CreateString(r.FSIPath)

	repofb.DatasetRefStart(builder)
	repofb.DatasetRefAddPeername(builder, peername)
	repofb.DatasetRefAddProfileID(builder, profileID)
	repofb.DatasetRefAddName(builder, name)
	repofb.DatasetRefAddPath(builder, path)
	repofb.DatasetRefAddFsiPath(builder, fsiPath)
	repofb.DatasetRefAddPublished(builder, r.Published)
	return repofb.DatasetRefEnd(builder)
}

// UnmarshalFlatbuffer decodes a job from a flatbuffer
func (r *DatasetRef) UnmarshalFlatbuffer(rfb *repofb.DatasetRef) (err error) {

	*r = DatasetRef{
		Peername:  string(rfb.Peername()),
		Name:      string(rfb.Name()),
		Path:      string(rfb.Path()),
		FSIPath:   string(rfb.FsiPath()),
		Published: rfb.Published(),
	}

	if pidstr := string(rfb.ProfileID()); pidstr != "" {
		r.ProfileID, err = profile.IDB58Decode(pidstr)
	}

	return err
}
