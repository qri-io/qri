package lib

import (
	"testing"
	"time"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
	"github.com/qri-io/registry/regserver/mock"
)

func TestRegistryRequests(t *testing.T) {
	var (
		movies, counter, cities, craigslist, sitemap repo.DatasetRef
	)
	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }
	defer func() {
		dsfs.Timestamp = prevTs
	}()

	reg := mock.NewMemRegistry()
	cli, _ := mock.NewMockServerRegistry(reg)
	mr, err := testrepo.NewTestRepo(cli)
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
		return
	}

	refs, err := mr.References(30, 0)
	if err != nil {
		t.Fatalf("error getting namespace: %s", err.Error())
	}

	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}
	req := NewRegistryRequests(node, nil)
	profile, err := node.Repo.Profile()
	if err != nil {
		t.Fatal(err)
	}

	// name refs and publish :)
	done := false
	for _, ref := range refs {
		ref.ProfileID = profile.ID
		ref.Published = true
		switch ref.Name {
		case "movies":
			movies = ref
		case "counter":
			counter = ref
		case "cities":
			cities = ref
		case "craigslist":
			craigslist = ref
		case "sitemap":
			sitemap = ref
		}
		if err := req.Publish(&ref, &done); err != nil {
			t.Fatal(err)
		}

	}

	// test getting a dataset from the registry
	citiesRef := repo.DatasetRef{
		Peername: "me",
		Name:     "cities",
	}
	citiesRes := repo.DatasetRef{}
	if err := req.GetDataset(&citiesRef, &citiesRes); err != nil {
		t.Error(err)
	}

	expect := "/map/QmSXfRRFy2T3gBARtvcf1GJAgg5dandb9fpqyUWAYzEcQq"
	if expect != citiesRes.Path {
		t.Errorf("error getting dataset from registry, expected path to be '%s', got %s", expect, citiesRes.Path)
	}
	if citiesRes.Dataset == nil {
		t.Errorf("error getting dataset from registry, dataset is nil")
	}
	if citiesRes.Published != true {
		t.Errorf("error getting dataset from registry, expected published to be 'true'")
	}

	// testing pagination
	cases := []struct {
		description string
		params      *RegistryListParams
		refs        []repo.DatasetRef
		err         string
	}{
		{"registry list - default", &RegistryListParams{}, []repo.DatasetRef{cities, counter, craigslist, movies, sitemap}, ""},
		{"registry list - negative offset and limit", &RegistryListParams{Offset: -1, Limit: -1}, []repo.DatasetRef{cities, counter, craigslist, movies, sitemap}, ""},
		{"registry list - happy path", &RegistryListParams{Offset: 0, Limit: 25}, []repo.DatasetRef{cities, counter, craigslist, movies, sitemap}, ""},
		{"registry list - offset 0 limit 3", &RegistryListParams{Offset: 0, Limit: 3}, []repo.DatasetRef{cities, counter, craigslist}, ""},
		{"registry list - offset 3 limit 3", &RegistryListParams{Offset: 3, Limit: 3}, []repo.DatasetRef{movies, sitemap}, ""},
		{"registry list - offset 6 limit 3", &RegistryListParams{Offset: 6, Limit: 3}, []repo.DatasetRef{}, ""},
	}
	for _, c := range cases {
		if err := req.List(c.params, &done); err != nil {
			t.Error(err)
		}

		got := c.params.Refs

		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case '%s' error mismatch: expected: %s, got: %s", c.description, c.err, err)
			continue
		}

		if c.err == "" && c.refs != nil {
			if len(c.refs) != len(got) {
				t.Errorf("case '%s' response length mismatch. expected %d, got: %d", c.description, len(c.refs), len(got))
				continue
			}

			for j, expect := range c.refs {
				// hack because registries aren't adding profileIDs
				got[j].ProfileID = expect.ProfileID
				if err := repo.CompareDatasetRef(expect, *got[j]); err != nil {
					t.Errorf("case '%s' expected dataset error. index %d mismatch: %s", c.description, j, err.Error())
					continue
				}
			}
		}

	}

	if err := req.Unpublish(&cities, &done); err != nil {
		t.Fatal(err)
	}

	rlp := &RegistryListParams{}

	if err := req.List(rlp, &done); err != nil {
		t.Error(err)
	}
	if len(rlp.Refs) != 4 {
		t.Errorf("expected registry to have 1 dataset. got: %d", reg.Datasets.Len())
	}
}
