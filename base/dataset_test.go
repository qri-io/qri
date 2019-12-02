package base

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/repo"
)

func TestListDatasets(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	// Limit to one
	res, err := ListDatasets(ctx, r, "", 1, 0, false, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 1 {
		t.Error("expected one dataset response")
	}

	// Limit to published datasets
	res, err = ListDatasets(ctx, r, "", 1, 0, false, true, false)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 0 {
		t.Error("expected no published datasets")
	}

	if err := SetPublishStatus(r, &ref, true); err != nil {
		t.Fatal(err)
	}

	// Limit to published datasets, after publishing cities
	res, err = ListDatasets(ctx, r, "", 1, 0, false, true, false)
	if err != nil {
		t.Error(err.Error())
	}

	if len(res) != 1 {
		t.Error("expected one published dataset response")
	}

	// Limit to datasets with "city" in their name
	res, err = ListDatasets(ctx, r, "city", 1, 0, false, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 0 {
		t.Error("expected no datasets with \"city\" in their name")
	}

	// Limit to datasets with "cit" in their name
	res, err = ListDatasets(ctx, r, "cit", 1, 0, false, false, false)
	if err != nil {
		t.Error(err.Error())
	}
	if len(res) != 1 {
		t.Error("expected one dataset with \"cit\" in their name")
	}
}

func TestFetchDataset(t *testing.T) {
	ctx := context.Background()
	r1 := newTestRepo(t)
	r2 := newTestRepo(t)
	ref := addCitiesDataset(t, r2)

	// Connect in memory Mapstore's behind the scene to simulate IPFS-like behavior.
	r1.Store().(*cafs.MapStore).AddConnection(r2.Store().(*cafs.MapStore))

	if err := FetchDataset(ctx, r1, &repo.DatasetRef{Peername: "foo", Name: "bar"}, true, true); err == nil {
		t.Error("expected add of invalid ref to error")
	}

	if err := FetchDataset(ctx, r1, &ref, true, true); err != nil {
		t.Error(err.Error())
	}
}

func TestDatasetPinning(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	if err := PinDataset(ctx, r, ref); err != nil {
		if err == repo.ErrNotPinner {
			t.Log("repo store doesn't support pinning")
		} else {
			t.Error(err.Error())
			return
		}
	}

	tc, err := dstest.NewTestCaseFromDir(testdataPath("counter"))
	if err != nil {
		t.Error(err.Error())
		return
	}

	ref2, err := CreateDataset(ctx, r, devNull, tc.Input, nil, false, false, false, true)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if err := PinDataset(ctx, r, ref2); err != nil && err != repo.ErrNotPinner {
		// TODO (b5) - not sure what's going on here
		t.Log(err.Error())
		return
	}

	if err := UnpinDataset(ctx, r, ref); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}

	if err := UnpinDataset(ctx, r, ref2); err != nil && err != repo.ErrNotPinner {
		t.Error(err.Error())
		return
	}
}

func TestRemoveNVersionsFromStore(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)

	bad := []struct {
		description string
		store       repo.Repo
		ref         *repo.DatasetRef
		n           int
		err         string
	}{
		{"No repo", nil, nil, 0, "need a repo"},
		{"No ref", r, nil, 0, "need a dataset reference with a path"},
		{"No ref.Path", r, &repo.DatasetRef{}, 0, "need a dataset reference with a path"},
		{"invalid n", r, &repo.DatasetRef{Path: "path", Dataset: &dataset.Dataset{}}, -2, "invalid 'n', n should be n >= 0 or n == -1 to indicate removing all versions"},
	}

	for _, c := range bad {
		_, err := RemoveNVersionsFromStore(ctx, c.store, c.ref, c.n)
		if err == nil {
			t.Errorf("case %s expected: '%s', got no error", c.description, c.err)
			continue
		}
		if c.err != err.Error() {
			t.Errorf("case %s error mismatch. expected: '%s', got: '%s'", c.description, c.err, err.Error())
		}
	}

	// create test repo and history
	// create history of 10 versions
	initDs := addCitiesDataset(t, r)
	refs := []*repo.DatasetRef{&initDs}
	historyTotal := 10
	for i := 2; i <= historyTotal; i++ {
		update := updateCitiesDataset(t, r, fmt.Sprintf("example city data version %d", i))
		refs = append(refs, &update)
	}

	good := []struct {
		description string
		n           int
	}{
		{"remove n == 0 versions", 0},
		{"remove n == 1 versions", 1},
		{"remove n == 3 versions", 3},
		// should not error when n is greater then the length of history
		{"remove n == 10 versions", 10},
	}

	for _, c := range good {
		// remove
		latestRef := refs[len(refs)-1]
		_, err := RemoveNVersionsFromStore(ctx, r, latestRef, c.n)
		if err != nil {
			t.Errorf("case '%s', unexpected err: %s", c.description, err.Error())
		}
		// verifyRefsRemoved will return an empty string
		// if the correct number of refs have been removed
		s := verifyRefsRemoved(ctx, r.Store(), refs, c.n)
		if s != "" {
			t.Errorf("case '%s', refs removed incorrectly: %s", c.description, s)
		}
		shorten := len(refs) - c.n
		if shorten < 0 {
			shorten = len(refs)
		}
		refs = refs[:shorten]
	}

	// remove the ds reference to the cities dataset before we populate
	// the repo with cities datasets again
	r.DeleteRef(initDs)

	// create test repo and history
	// create history of 10 versions
	initDs = addCitiesDataset(t, r)
	refs = []*repo.DatasetRef{&initDs}
	for i := 2; i <= historyTotal; i++ {
		update := updateCitiesDataset(t, r, fmt.Sprintf("example city data version %d", i))
		refs = append(refs, &update)
	}
	_, err := RemoveNVersionsFromStore(ctx, r, refs[len(refs)-1], -1)
	if err != nil {
		t.Errorf("case 'remove all', unexpected err: %s", err.Error())
	}
	s := verifyRefsRemoved(ctx, r.Store(), refs, len(refs))
	if s != "" {
		t.Errorf("case 'remove all', refs removed incorrectly: %s", s)
	}

}

// takes store s, where datasets have been added/removed
// takes a list of refs, where refs[0] is the initial (oldest) dataset
// take int n where n is the number of MOST RECENT datasets that should
// have been removed
// assumes that each Dataset has a Meta component with a Title
func verifyRefsRemoved(ctx context.Context, s cafs.Filestore, refs []*repo.DatasetRef, n int) string {

	// datasets from index len(refs) - n - 1 SHOULD EXISTS
	// we should error if they DON't exist
	errString := ""
	for i, ref := range refs {
		// datasets from index len(refs) - 1 to len(refs) - n SHOULD NOT EXISTS
		// we should error if they exist

		exists, err := s.Has(ctx, ref.Path)
		if err != nil {
			return fmt.Sprintf("error checking ref '%s' with title '%s' from store", ref.Dataset.Path, ref.Dataset.Meta.Title)
		}

		// datasets that are less then len(refs) - n, should exist
		if i < len(refs)-n {
			if exists == true {
				continue
			}
			errString += fmt.Sprintf("\nref '%s' should exist in the store, but does NOT", ref.Dataset.Meta.Title)
			continue
		}

		// datasets that are greater then len(refs) - n, should NOT exist
		if exists == false {
			continue
		}
		errString += fmt.Sprintf("\nref '%s' should NOT exist in the store, but does", ref.Dataset.Meta.Title)

	}
	return errString
}

func TestVerifyRefsRemove(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	// create test repo and history
	// create history of 10 versions
	initDs := addCitiesDataset(t, r)

	//
	refs := []*repo.DatasetRef{&initDs}
	historyTotal := 3
	for i := 2; i <= historyTotal; i++ {
		update := updateCitiesDataset(t, r, fmt.Sprintf("example city data version %d", i))
		refs = append(refs, &update)
	}
	// test that all real refs exist
	// aka n = 0
	s := verifyRefsRemoved(ctx, r.Store(), refs, 0)
	if s != "" {
		t.Errorf("case 'all refs should exist' should return empty string, got '%s'", s)
	}

	// test that when we have refs in the store
	// but we say that there should be no refs in the store
	// we get the proper response:
	s = verifyRefsRemoved(ctx, r.Store(), refs, 2)
	sExpected := "\nref 'example city data version 2' should NOT exist in the store, but does\nref 'example city data version 3' should NOT exist in the store, but does"
	if s != sExpected {
		t.Errorf("case 'all refs exist, but we say 2 should not' response mismatch: expected '%s', got '%s'", sExpected, s)
	}

	for i := 0; i < 3; i++ {
		fakeRef := repo.DatasetRef{
			Path: fmt.Sprintf("/map/%d", i),
			Dataset: &dataset.Dataset{
				Meta: &dataset.Meta{
					Title: fmt.Sprintf("Fake Ref version %d", i),
				},
			},
		}
		refs = append(refs, &fakeRef)
	}
	// test that all real refs exist in store
	// and all fake refs do not exist in store
	// aka n = 3
	s = verifyRefsRemoved(ctx, r.Store(), refs, 3)
	if s != "" {
		t.Errorf("case '3 fake refs, with n == 3' should return empty string, got '%s'", s)
	}

	// test that when we say we do have refs in the store
	// but we really don't, we get the proper response:
	s = verifyRefsRemoved(ctx, r.Store(), refs, 0)
	sExpected = `
ref 'Fake Ref version 0' should exist in the store, but does NOT
ref 'Fake Ref version 1' should exist in the store, but does NOT
ref 'Fake Ref version 2' should exist in the store, but does NOT`
	if s != sExpected {
		t.Errorf("case 'expect empty refs to exist' response mismatch: expected '%s', got '%s'", sExpected, s)
	}
}

func TestRawDatasetRefs(t *testing.T) {
	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	minute := 0
	dsfs.Timestamp = func() time.Time {
		minute++
		return time.Date(2001, 01, 01, 01, minute, 01, 01, time.UTC)
	}

	ctx := context.Background()
	r := newTestRepo(t)
	addCitiesDataset(t, r)

	actual, err := RawDatasetRefs(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	expect := `0 Peername:  peer
  ProfileID: 9tmwSYB7dPRUXaEwJRNgzb6NbwPYNXrYyeahyHPAUqrTYd3Z6bVS9z1mCDsRmvb
  Name:      cities
  Path:      /map/QmbU34XVYPGeEGjJ93rBm4Nac2g4hBYFouDnu9p9psccDB
  FSIPath:   
  Published: false
`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
