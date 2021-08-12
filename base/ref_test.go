package base

import (
	"context"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

func TestInAuthorNamespace(t *testing.T) {
	r := newTestRepo(t)
	author := r.Profiles().Owner()
	ctx := context.Background()
	ref := addCitiesDataset(t, r)

	if !InAuthorNamespace(ctx, author, ref) {
		t.Errorf("expected %s true", ref.String())
	}

	ref = dsref.Ref{}
	if InAuthorNamespace(ctx, author, ref) {
		t.Errorf("expected %s false", ref.String())
	}

	ref = dsref.Ref{ProfileID: "fake"}
	if InAuthorNamespace(ctx, author, ref) {
		t.Errorf("expected %s false", ref.String())
	}
}

func TestSetPublishStatus(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	ref := addCitiesDataset(t, r)
	author := r.Profiles().Owner()

	if err := SetPublishStatus(ctx, r, author, ref, true); err != nil {
		t.Error(err)
	}

	res, err := repo.GetVersionInfoShim(r, ref)
	if err != nil {
		t.Fatal(err)
	}
	if res.Published != true {
		t.Errorf("expected published to equal true: %v,%v", ref, res)
	}

	if err := SetPublishStatus(ctx, r, author, ref, false); err != nil {
		t.Error(err)
	}
	res, err = repo.GetVersionInfoShim(r, ref)
	if err != nil {
		t.Fatal(err)
	}
	if res.Published != false {
		t.Errorf("expected published to equal false: %v,%v", ref, res)
	}

	if err := SetPublishStatus(ctx, r, author, dsref.Ref{Name: "foo"}, false); err == nil {
		t.Error("expected invalid reference to error")
	}

	outside := dsref.MustParse("a/b@QmX1oSPMbzkhk33EutuadL4sqsivsRKmMx5hAnZL2mRAM1/ipfs/Qmd")
	vi := dsref.NewVersionInfoFromRef(outside)
	if err := repo.PutVersionInfoShim(ctx, r, &vi); err != nil {
		t.Fatal(err)
	}

	r.Profiles().PutProfile(&profile.Profile{ID: profile.IDB58DecodeOrEmpty(outside.ProfileID), Peername: outside.Username})

	if err := SetPublishStatus(ctx, r, author, outside, true); err == nil {
		t.Error("expected setting the publish status of a name outside peer's namespace to fail")
	}
}

func TestReplaceRefIfMoreRecent(t *testing.T) {
	r := newTestRepo(t)
	older := time.Date(2019, 1, 1, 12, 0, 0, 0, time.UTC)
	newer := older.AddDate(1, 0, 0)
	cases := []struct {
		description string
		a, b        reporef.DatasetRef
		path        string
	}{
		{
			"first dataset is older then the second",
			reporef.DatasetRef{
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
			reporef.DatasetRef{
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
			reporef.DatasetRef{
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
			reporef.DatasetRef{
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
			reporef.DatasetRef{
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
			reporef.DatasetRef{
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
		ref, err := r.GetRef(reporef.DatasetRef{Peername: c.a.Peername, Name: c.a.Name})
		if err != nil {
			t.Fatal(err)
		}
		if ref.Path != c.path {
			t.Errorf("case '%s', ref path error, expected: '%s', got: '%s'", c.description, c.path, ref.Path)
		}
	}

	casesError := []struct {
		description string
		a, b        reporef.DatasetRef
		err         string
	}{
		{
			"original ref has no timestamp & should error",
			reporef.DatasetRef{
				Peername:  "woo",
				Name:      "err",
				Path:      "/map/first",
				ProfileID: "id",
			},
			reporef.DatasetRef{
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
			reporef.DatasetRef{
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
			reporef.DatasetRef{},
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
