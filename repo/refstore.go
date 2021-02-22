package repo

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/dsref"
	reporef "github.com/qri-io/qri/repo/ref"
)

// Refstore keeps a collection of dataset references, Refstores require complete
// references (with both alias and identifiers), and can carry only one of a
// given alias eg: putting peer/dataset@a/ipfs/b when a ref with alias peer/dataset
// is already in the store will overwrite the stored reference
type Refstore interface {
	// PutRef adds a reference to the store. References must be complete with
	// Peername, Name, and Path/FSIPath specified
	PutRef(ref reporef.DatasetRef) error
	// GetRef "completes" a passed in alias (reporef.DatasetRef with at least Peername
	// and Name field specified), filling in missing fields with a stored ref
	GetRef(ref reporef.DatasetRef) (reporef.DatasetRef, error)
	// DeleteRef removes a reference from the store
	DeleteRef(ref reporef.DatasetRef) error
	// References returns a set of references from the store
	References(offset, limit int) ([]reporef.DatasetRef, error)
	// RefCount returns the number of references in the store
	RefCount() (int, error)
}

// ListVersionInfoShim wraps a call to References, converting results to a
// slice of VersionInfos
func ListVersionInfoShim(r Repo, offset, limit int) ([]dsref.VersionInfo, error) {
	refs, err := r.References(offset, limit)
	if err != nil {
		return nil, err
	}

	res := make([]dsref.VersionInfo, len(refs))
	for i, rref := range refs {
		res[i] = reporef.ConvertToVersionInfo(&rref)
	}
	return res, nil
}

// GetVersionInfoShim is a shim for getting away from the old
// DatasetRef, CanonicalizeDatasetRef, RefStore stack
func GetVersionInfoShim(r Repo, ref dsref.Ref) (*dsref.VersionInfo, error) {
	rref, err := r.GetRef(reporef.RefFromDsref(ref))
	if err != nil {
		return nil, err
	}
	vi := reporef.ConvertToVersionInfo(&rref)
	return &vi, nil
}

// PutVersionInfoShim is a shim for getting away from old stack of
// DatasetRef, CanonicalizeDatasetRef, and RefStores
// while still safely interacting with the repo.Refstore API
func PutVersionInfoShim(ctx context.Context, r Repo, vi *dsref.VersionInfo) error {
	// attempt to look up peerIDs when not set
	if vi.ProfileID == "" && vi.Username != "" {
		rref := &reporef.DatasetRef{Peername: vi.Username}
		if err := canonicalizeProfile(ctx, r, rref); err == nil {
			vi.ProfileID = rref.ProfileID.String()
		}
	}
	return r.PutRef(reporef.RefFromVersionInfo(vi))
}

// DeleteVersionInfoShim is a shim for getting away from the old stack of
// DatasetRef, CanonicalizeDatasetRef, and RefStore
// while still safely interacting with the repo.Refstore API
func DeleteVersionInfoShim(ctx context.Context, r Repo, ref dsref.Ref) (*dsref.VersionInfo, error) {
	rref := reporef.RefFromDsref(ref)
	if err := canonicalizeDatasetRef(ctx, r, &rref); err != nil && err != ErrNoHistory {
		return nil, err
	}
	if err := r.DeleteRef(rref); err != nil {
		return nil, err
	}
	vi := reporef.ConvertToVersionInfo(&rref)
	return &vi, nil
}

// ListDatasetsShim gets away from the old stack of DatasetRef,
// CanonicalizeDatasetRef and RefStore
func ListDatasetsShim(r Repo, offset, limit int) ([]dsref.VersionInfo, error) {
	rrefs, err := r.References(offset, limit)
	if err != nil {
		return nil, err
	}

	vis := make([]dsref.VersionInfo, 0, limit)
	for _, ref := range rrefs {
		if offset > 0 {
			offset--
			continue
		}
		vis = append(vis, reporef.ConvertToVersionInfo(&ref))
		if len(vis) == limit {
			return vis, nil
		}
	}

	return vis, nil
}

// canonicalizeDatasetRef uses the user's repo to turn any local aliases into full dataset
// references using known canonical peernames and paths. If the provided reference is not
// in the local repo, still do the work of handling aliases, but return a repo.ErrNotFound
// error, which callers can respond to by possibly contacting remote repos.
func canonicalizeDatasetRef(ctx context.Context, r Repo, ref *reporef.DatasetRef) error {
	if ref.IsEmpty() {
		return ErrEmptyRef
	}

	if err := canonicalizeProfile(ctx, r, ref); err != nil {
		return err
	}

	got, err := r.GetRef(*ref)
	if err != nil {
		return err
	}

	// TODO (b5) - this is the assign pattern, refactor into a method on reporef.DatasetRef
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
	if ref.ProfileID != got.ProfileID || ref.Name != got.Name {
		return fmt.Errorf("Given datasetRef %s does not match datasetRef on file: %s", ref.String(), got.String())
	}

	if got.Path == "" {
		return ErrNoHistory
	}

	return nil
}

// canonicalizeProfile populates dataset reporef.DatasetRef ProfileID and Peername properties,
// changing aliases to known names, and adding ProfileID from a peerstore
func canonicalizeProfile(ctx context.Context, r Repo, ref *reporef.DatasetRef) error {
	if ref.Peername == "" && ref.ProfileID == "" {
		return nil
	}

	p := r.Profiles().Owner()

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
func CompareDatasetRef(a, b reporef.DatasetRef) error {
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
