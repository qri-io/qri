package repo

import (
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
	// Peername, Name, and Path specified
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

// DeleteVersionInfoShim is a shim for getting away from the old stack of
// DatasetRef, CanonicalizeDatasetRef, and RefStores
// while still safely interacting with the repo.Refstore API
func DeleteVersionInfoShim(r Repo, ref dsref.Ref) (*dsref.VersionInfo, error) {
	rref := reporef.RefFromDsref(ref)
	if err := CanonicalizeDatasetRef(r, &rref); err != nil && err != ErrNoHistory {
		return nil, err
	}
	if err := r.DeleteRef(rref); err != nil {
		return nil, err
	}
	vi := reporef.ConvertToVersionInfo(&rref)
	return &vi, nil
}

// PutVersionInfoShim is a shim for getting away from old stack of
// DatasetRef, CanonicalizeDatasetRef, and RefStores
// while still safely interacting with the repo.Refstore API
func PutVersionInfoShim(r Repo, vi *dsref.VersionInfo) error {
	return r.PutRef(reporef.RefFromVersionInfo(vi))
}

// TODO(dlong): In the near future, switch to a new utility that resolves references to specific
// versions by using logbook. A ref should resolve to a pair of (init-id, head-ref), where the
// init-id is the stable unchanging identifier for a dataset (derived from logbook) and head-ref
// is the current head version. Use that everywhere in the code, instead of CanonicalizeDatasetRef.

// CanonicalizeDatasetRef uses the user's repo to turn any local aliases into full dataset
// references using known canonical peernames and paths. If the provided reference is not
// in the local repo, still do the work of handling aliases, but return a repo.ErrNotFound
// error, which callers can respond to by possibly contacting remote repos.
func CanonicalizeDatasetRef(r Repo, ref *reporef.DatasetRef) error {
	if ref.IsEmpty() {
		return ErrEmptyRef
	}

	if err := CanonicalizeProfile(r, ref); err != nil {
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

// CanonicalizeProfile populates dataset reporef.DatasetRef ProfileID and Peername properties,
// changing aliases to known names, and adding ProfileID from a peerstore
func CanonicalizeProfile(r Repo, ref *reporef.DatasetRef) error {
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
