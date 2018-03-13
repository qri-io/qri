package test

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/qri/repo"
)

func runTestRefstore(r repo.Repo) error {
	for _, test := range []RepoTestFunc{
		testInvalidRefs,
		testRefs,
		testRefstore,
	} {
		if err := test(r); err != nil {
			return fmt.Errorf("RunTestRefstore: %s", err.Error())
		}
	}

	return nil
}

func testInvalidRefs(r repo.Repo) error {
	err := r.PutRef(repo.DatasetRef{Name: "a", Path: "/path/to/a/thing"})
	if err != repo.ErrPeerIDRequired {
		return fmt.Errorf("attempting to put empty peerID in refstore should return repo.ErrPeerIDRequired, got: %s", err)
	}

	err = r.PutRef(repo.DatasetRef{PeerID: "peerID", Peername: "peer", Path: "/path/to/a/thing"})
	if err != repo.ErrNameRequired {
		return fmt.Errorf("attempting to put empty name in refstore should return repo.ErrNameRequired, got: %s", err)
	}

	err = r.PutRef(repo.DatasetRef{PeerID: "peerID", Peername: "peer", Name: "a", Path: ""})
	if err != repo.ErrPathRequired {
		return fmt.Errorf("attempting to put empty path in refstore should return repo.ErrPathRequired, got: %s", err)
	}

	return nil
}

func testRefs(r repo.Repo) error {
	path, err := r.Store().Put(cafs.NewMemfileBytes("test", []byte(`{ "title": "test data" }`)), true)
	if err != nil {
		return fmt.Errorf("error putting test file in datastore: %s", err.Error())
	}

	ref := repo.DatasetRef{PeerID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", Name: "test", Path: path.String(), Peername: "peer"}

	if err := r.PutRef(ref); err != nil {
		return fmt.Errorf("repo.PutName: %s", err.Error())
	}

	res, err := r.GetRef(repo.DatasetRef{PeerID: ref.PeerID, Name: ref.Name})
	if err != nil {
		return fmt.Errorf("repo.GetRef with peerID/name: %s, ref: %s", err.Error(), repo.DatasetRef{PeerID: ref.PeerID, Name: ref.Name})
	}
	if !ref.Equal(res) {
		return fmt.Errorf("repo.GetRef with peerID/name response mistmatch. expected: %s, got: %s", ref, res)
	}

	res, err = r.GetRef(repo.DatasetRef{Path: ref.Path})
	if err != nil {
		return fmt.Errorf("repo.GetRef with path: %s", err.Error())
	}
	if !ref.Equal(res) {
		return fmt.Errorf("repo.GetRef with path response mismatch. expected: %s, got: %s", ref, res)
	}

	if err := r.DeleteRef(ref); err != nil {
		return fmt.Errorf("repo.DeleteName: %s", err.Error())
	}

	_, err = r.GetRef(ref)
	if err != repo.ErrNotFound {
		return fmt.Errorf("repo.GetRef where ref is deleted should return ErrNotFound")
	}
	err = nil

	if err := r.Store().Delete(datastore.NewKey(ref.Path)); err != nil {
		return fmt.Errorf("error removing file from store")
	}
	return nil
}

func testRefstore(r repo.Repo) error {
	aname := "test_namespace_a"
	bname := "test_namespace_b"
	refs := []repo.DatasetRef{
		{PeerID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", Peername: "peer", Name: aname},
		{PeerID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", Peername: "peer", Name: bname},
		{PeerID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", Peername: "peer", Name: "test_namespace_c"},
		{PeerID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", Peername: "peer", Name: "test_namespace_d"},
		{PeerID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt", Peername: "peer", Name: "test_namespace_e"},
	}
	for _, ref := range refs {
		path, err := r.Store().Put(cafs.NewMemfileBytes("test", []byte(fmt.Sprintf(`{ "title": "test_dataset_%s" }`, ref.Name))), true)
		if err != nil {
			return fmt.Errorf("error putting test file in datastore: %s", err.Error())
		}
		ref.Path = path.String()
		if err := r.PutRef(ref); err != nil {
			return fmt.Errorf("error putting name in repo for namespace test: %s", err.Error())
		}
	}

	count, err := r.RefCount()
	if err != nil {
		return fmt.Errorf("repo.NameCount: %s", err.Error())
	}
	if count < 5 {
		return fmt.Errorf("repo.NameCount should have returned at least 5 results")
	}

	names := []repo.DatasetRef{}
	pages := count
	pageSize := count / pages
	for i := 0; i <= pages; i++ {
		res, err := r.References(pageSize, i*pageSize)
		if err != nil {
			return fmt.Errorf("repo.References(%d,%d): %s", pageSize, i*pageSize, err.Error())
		}
		names = append(names, res...)
	}
	if len(names) != count {
		return fmt.Errorf("failed to read all paginated names. expected %d results, got %d", count, len(names))
	}

	idxs := map[string]int{}
	for i, ref := range names {
		idxs[ref.Name] = i
	}
	for i, ref := range refs {
		if i > 0 {
			if idxs[ref.Name] < idxs[refs[i-1].Name] {
				return fmt.Errorf("expected results to be returned in lexographical order. %s:%d, %s:%d", ref.Name, idxs[ref.Name], refs[i-1].Name, idxs[refs[i-1].Name])
			}
		}
	}

	for _, ref := range refs {
		if err := r.Store().Delete(datastore.NewKey(ref.Path)); err != nil {
			return fmt.Errorf("error removing path from repo store: %s", err.Error())
		}
		if err := r.DeleteRef(ref); err != nil {
			return fmt.Errorf("error removing test name from namespace: %s", err.Error())
		}
	}
	return nil
}
