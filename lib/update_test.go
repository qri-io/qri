package lib

import (
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/cron"
	"github.com/qri-io/qri/repo"
)

func TestDatasetMethodsRun(t *testing.T) {
	node := newTestQriNode(t)
	inst := &Instance{node: node}

	m := NewUpdateMethods(inst)
	res := &repo.DatasetRef{}
	if err := m.Run(&Job{Name: "me/bad_dataset", Type: cron.JTDataset}, res); err == nil {
		t.Error("expected update to nonexistent dataset to error")
	}

	ref := addNowTransformDataset(t, node)
	res = &repo.DatasetRef{}
	if err := m.Run(&Job{Name: ref.AliasString(), Type: cron.JTDataset /* Recall: "tf", ReturnBody: true */}, res); err != nil {
		t.Errorf("update error: %s", err)
	}

	dsm := NewDatasetRequests(inst.node, nil)
	// run a manual save to lose the transform
	err := dsm.Save(&SaveParams{Dataset: &dataset.Dataset{
		Peername: res.Peername,
		Name:     res.Name,
		Meta:     &dataset.Meta{Title: "an updated title"},
	}}, res)
	if err != nil {
		t.Error("save failed")
	}

	// update should grab the transform from 2 commits back
	if err := m.Run(&Job{Name: res.AliasString(), Type: cron.JTDataset /* ReturnBody: true */}, res); err != nil {
		t.Error(err)
	}
}
