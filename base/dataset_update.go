package base

import (
	"fmt"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo"
)

// UpdateDataset re-runs a dataset with a given transform, creating a new verion in the process
func UpdateDataset(r repo.Repo, ref repo.DatasetRef, commit *dataset.CommitPod) (update repo.DatasetRef, err error) {
	if err = ReadDataset(r, &ref); err != nil {
		return
	}

	ds := ref.Dataset
	if ds.Transform == nil {
		err = fmt.Errorf("transform script is required to automate updates to your own datasets")
		return
	}

	return
}
