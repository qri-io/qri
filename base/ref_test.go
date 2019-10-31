package base

import (
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestInLocalNamespace(t *testing.T) {
	r := newTestRepo(t)
	cities := addCitiesDataset(t, r)
	ref := &cities

	if !InLocalNamespace(r, ref) {
		t.Errorf("expected %s true", ref.String())
	}

	ref = &repo.DatasetRef{}
	if InLocalNamespace(r, ref) {
		t.Errorf("expected %s false", ref.String())
	}

	ref = &repo.DatasetRef{ProfileID: profile.ID("fake")}
	if InLocalNamespace(r, ref) {
		t.Errorf("expected %s false", ref.String())
	}
}

func TestSetPublishStatus(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	if err := SetPublishStatus(r, &ref, true); err != nil {
		t.Error(err)
	}
	res, err := r.GetRef(repo.DatasetRef{Peername: ref.Peername, Name: ref.Name})
	if err != nil {
		t.Fatal(err)
	}
	if res.Published != true {
		t.Errorf("expected published to equal true: %s,%s", ref, res)
	}

	if err := SetPublishStatus(r, &ref, false); err != nil {
		t.Error(err)
	}
	res, err = r.GetRef(repo.DatasetRef{Peername: ref.Peername, Name: ref.Name})
	if err != nil {
		t.Fatal(err)
	}
	if res.Published != false {
		t.Errorf("expected published to equal false: %s,%s", ref, res)
	}

	if err := SetPublishStatus(r, &repo.DatasetRef{Name: "foo"}, false); err == nil {
		t.Error("expected invalid reference to error")
	}

	outside := repo.MustParseDatasetRef("a/b@QmX1oSPMbzkhk33EutuadL4sqsivsRKmMx5hAnZL2mRAM1/ipfs/d")
	if err := r.PutRef(outside); err != nil {
		t.Fatal(err)
	}

	r.Profiles().PutProfile(&profile.Profile{ID: outside.ProfileID, Peername: outside.Peername})

	if err := SetPublishStatus(r, &outside, true); err == nil {
		t.Error("expected setting the publish status of a name outside peer's namespace to fail")
	}
}

func TestReplaceRefIfMoreRecent(t *testing.T) {
	r := newTestRepo(t)
	older := time.Date(2019, 1, 1, 12, 0, 0, 0, time.UTC)
	newer := older.AddDate(1, 0, 0)
	cases := []struct {
		description string
		a, b        repo.DatasetRef
		path        string
	}{
		{
			"first dataset is older then the second",
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_older",
				Path:      "/map/first",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: older,
					},
				},
			},
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_older",
				Path:      "/map/second",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			"/map/second",
		},
		{
			"first dataset is newer then the second",
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_newer",
				Path:      "/map/first",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_newer",
				Path:      "/map/second",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: older,
					},
				},
			},
			"/map/first",
		},
		{
			"first dataset is same time as the the second",
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_same",
				Path:      "/map/first",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "first_same",
				Path:      "/map/second",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			"/map/second",
		},
	}

	for _, c := range cases {
		if err := r.PutRef(c.a); err != nil {
			t.Fatal(err)
		}
		if err := ReplaceRefIfMoreRecent(r, &c.a, &c.b); err != nil {
			t.Fatal(err)
		}
		ref, err := r.GetRef(repo.DatasetRef{Peername: c.a.Peername, Name: c.a.Name})
		if err != nil {
			t.Fatal(err)
		}
		if ref.Path != c.path {
			t.Errorf("case '%s', ref path error, expected: '%s', got: '%s'", c.description, c.path, ref.Path)
		}
	}

	casesError := []struct {
		description string
		a, b        repo.DatasetRef
		err         string
	}{
		{
			"original ref has no timestamp & should error",
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "err",
				Path:      "/map/first",
				ProfileID: "id",
			},
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "err",
				Path:      "/map/second",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			"previous dataset ref is not fully derefernced",
		},
		{
			"added ref has no timestamp & should error",
			repo.DatasetRef{
				Peername:  "woo",
				Name:      "err",
				Path:      "/map/first",
				ProfileID: "id",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: newer,
					},
				},
			},
			repo.DatasetRef{},
			"added dataset ref is not fully dereferenced",
		},
	}

	for _, c := range casesError {
		if err := r.PutRef(c.a); err != nil {
			t.Fatal(err)
		}
		err := ReplaceRefIfMoreRecent(r, &c.a, &c.b)
		if err == nil {
			t.Errorf("case '%s' did not error", c.description)
		}
		if err.Error() != c.err {
			t.Errorf("case '%s', error mismatch. expected: '%s', got: '%s'", c.description, c.err, err.Error())
		}
	}
}
