package base

import (
	"testing"

	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func TestInLocalNamespace(t *testing.T) {
	r := newTestRepo(t)
	reff := addCitiesDataset(t, r)
	ref := &reff

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

	ref.Published = true
	if err := SetPublishStatus(r, &ref); err != nil {
		t.Error(err)
	}
	res, err := r.GetRef(repo.DatasetRef{Peername: ref.Peername, Name: ref.Name})
	if err != nil {
		t.Fatal(err)
	}
	if res.Published != true {
		t.Errorf("expected published to equal true: %s,%s", ref, res)
	}

	ref.Published = false
	if err := SetPublishStatus(r, &ref); err != nil {
		t.Error(err)
	}
	res, err = r.GetRef(repo.DatasetRef{Peername: ref.Peername, Name: ref.Name})
	if err != nil {
		t.Fatal(err)
	}
	if res.Published != false {
		t.Errorf("expected published to equal false: %s,%s", ref, res)
	}

	if err := SetPublishStatus(r, &repo.DatasetRef{Name: "foo"}); err == nil {
		t.Error("expected invalid reference to error")
	}

	outside := repo.MustParseDatasetRef("a/b@QmX1oSPMbzkhk33EutuadL4sqsivsRKmMx5hAnZL2mRAM1/ipfs/d")
	if err := r.PutRef(outside); err != nil {
		t.Fatal(err)
	}

	r.Profiles().PutProfile(&profile.Profile{ID: outside.ProfileID, Peername: outside.Peername})

	outside.Published = true
	if err := SetPublishStatus(r, &outside); err == nil {
		t.Error("expected setting the publish status of a name outside peer's namespace to fail")
	}
}
