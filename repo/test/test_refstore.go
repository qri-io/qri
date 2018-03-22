package test

import (
	"fmt"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func testRefstore(t *testing.T, rmf RepoMakerFunc) {
	for _, test := range []repoTestFunc{
		testInvalidRefs,
		testRefs,
		testRefstoreMain,
	} {
		test(t, rmf)
	}
}

func testInvalidRefs(t *testing.T, rmf RepoMakerFunc) {
	r := rmf(t)
	err := r.PutRef(repo.DatasetRef{Name: "a", Path: "/path/to/a/thing"})
	if err != repo.ErrPeerIDRequired {
		t.Errorf("attempting to put empty peerID in refstore should return repo.ErrPeerIDRequired, got: %s", err)
		return
	}

	err = r.PutRef(repo.DatasetRef{ProfileID: profile.ID("badProfileID"), Peername: "peer", Path: "/path/to/a/thing"})
	if err != repo.ErrNameRequired {
		t.Errorf("attempting to put empty name in refstore should return repo.ErrNameRequired, got: %s", err)
		return
	}

	err = r.PutRef(repo.DatasetRef{ProfileID: profile.ID("badProfileID"), Peername: "peer", Name: "a", Path: ""})
	if err != repo.ErrPathRequired {
		t.Errorf("attempting to put empty path in refstore should return repo.ErrPathRequired, got: %s", err)
		return
	}

	return
}

func testRefs(t *testing.T, rmf RepoMakerFunc) {
	r := rmf(t)
	path, err := r.Store().Put(cafs.NewMemfileBytes("test", []byte(`{ "title": "test data" }`)), true)
	if err != nil {
		t.Errorf("error putting test file in datastore: %s", err.Error())
		return
	}

	ref := repo.DatasetRef{ProfileID: profile.IDB58MustDecode("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"), Name: "test", Path: path.String(), Peername: "peer"}

	if err := r.PutRef(ref); err != nil {
		t.Errorf("repo.PutName: %s", err.Error())
		return
	}

	res, err := r.GetRef(repo.DatasetRef{ProfileID: ref.ProfileID, Name: ref.Name})
	if err != nil {
		t.Errorf("repo.GetRef with peerID/name: %s, ref: %s", err.Error(), repo.DatasetRef{ProfileID: ref.ProfileID, Name: ref.Name})
		return
	}
	if !ref.Equal(res) {
		t.Errorf("repo.GetRef with peerID/name response mistmatch. expected: %s, got: %s", ref, res)
		return
	}

	res, err = r.GetRef(repo.DatasetRef{Path: ref.Path})
	if err != nil {
		t.Errorf("repo.GetRef with path: %s", err.Error())
		return
	}
	if !ref.Equal(res) {
		t.Errorf("repo.GetRef with path response mismatch. expected: %s, got: %s", ref, res)
		return
	}

	if err := r.DeleteRef(ref); err != nil {
		t.Errorf("repo.DeleteName: %s", err.Error())
		return
	}

	_, err = r.GetRef(ref)
	if err != repo.ErrNotFound {
		t.Errorf("repo.GetRef where ref is deleted should return ErrNotFound")
		return
	}
	err = nil

	if err := r.Store().Delete(datastore.NewKey(ref.Path)); err != nil {
		t.Errorf("error removing file from store")
		return
	}
	return
}

func testRefstoreMain(t *testing.T, rmf RepoMakerFunc) {
	r := rmf(t)
	aname := "test_namespace_a"
	bname := "test_namespace_b"
	refs := []repo.DatasetRef{
		{ProfileID: profile.IDB58MustDecode("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"), Peername: "peer", Name: aname},
		{ProfileID: profile.IDB58MustDecode("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"), Peername: "peer", Name: bname},
		{ProfileID: profile.IDB58MustDecode("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"), Peername: "peer", Name: "test_namespace_c"},
		{ProfileID: profile.IDB58MustDecode("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"), Peername: "peer", Name: "test_namespace_d"},
		{ProfileID: profile.IDB58MustDecode("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"), Peername: "peer", Name: "test_namespace_e"},
	}
	for _, ref := range refs {
		path, err := r.Store().Put(cafs.NewMemfileBytes("test", []byte(fmt.Sprintf(`{ "title": "test_dataset_%s" }`, ref.Name))), true)
		if err != nil {
			t.Errorf("error putting test file in datastore: %s", err.Error())
			return
		}
		ref.Path = path.String()
		if err := r.PutRef(ref); err != nil {
			t.Errorf("error putting name in repo for namespace test: %s", err.Error())
			return
		}
	}

	count, err := r.RefCount()
	if err != nil {
		t.Errorf("repo.NameCount: %s", err.Error())
		return
	}
	if count < 5 {
		t.Errorf("repo.NameCount should have returned at least 5 results")
		return
	}

	names := []repo.DatasetRef{}
	pages := count
	pageSize := count / pages
	for i := 0; i <= pages; i++ {
		res, err := r.References(pageSize, i*pageSize)
		if err != nil {
			t.Errorf("repo.References(%d,%d): %s", pageSize, i*pageSize, err.Error())
			return
		}
		names = append(names, res...)
	}
	if len(names) != count {
		t.Errorf("failed to read all paginated names. expected %d results, got %d", count, len(names))
		return
	}

	idxs := map[string]int{}
	for i, ref := range names {
		idxs[ref.Name] = i
	}
	for i, ref := range refs {
		if i > 0 {
			if idxs[ref.Name] < idxs[refs[i-1].Name] {
				t.Errorf("expected results to be returned in lexographical order. %s:%d, %s:%d", ref.Name, idxs[ref.Name], refs[i-1].Name, idxs[refs[i-1].Name])
				return
			}
		}
	}

	for _, ref := range refs {
		if err := r.Store().Delete(datastore.NewKey(ref.Path)); err != nil {
			t.Errorf("error removing path from repo store: %s", err.Error())
			return
		}
		if err := r.DeleteRef(ref); err != nil {
			t.Errorf("error removing test name from namespace: %s", err.Error())
			return
		}
	}
	return
}
