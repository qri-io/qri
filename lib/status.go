package lib

import (
	"github.com/qri-io/qri/fsi"
)

// StatusItem is an alias for an fsi.StatusItem
type StatusItem = fsi.StatusItem

// Status checks for any modifications or errors in a linked directory
func (r *DatasetRequests) Status(dir *string, res *[]StatusItem) (err error) {
	if r.cli != nil {
		return r.cli.Call("DatasetRequests.Status", dir, res)
	}

	fsint := fsi.NewFSI(r.node.Repo, "")
	*res, err = fsint.Status(*dir)
	return err
}