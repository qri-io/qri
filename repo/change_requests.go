package repo

import (
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
)

// ChangeRequestStore is the interface for storying & manipulating Change Requests
// Qri repos should embed a ChangeRequestStore
type ChangeRequestStore interface {
	// Put a change request into the store
	PutChangeRequest(path datastore.Key, cr *ChangeRequest) error
	// Get a change request by it's path
	GetChangeRequest(path datastore.Key) (*ChangeRequest, error)
	// accept an open change request
	// AcceptChangeRequest(path datastore.Key) error
	// decline an open change request
	// DeclineChangeRequest(path datastore.Key) error
	// get change requests for a given target
	ChangeRequestsForTarget(target datastore.Key, limit, offset int) ([]*ChangeRequest, error)
	// list change requests in this store
	ListChangeRequests(limit, offset int) ([]*ChangeRequest, error)
}

const (
	// ChangeRequestStatusOpen is a request that hasn't been addressed yet
	ChangeRequestStatusOpen = "open"
	// ChangeRequestStatusAccepted have been merged into
	// the target history
	ChangeRequestStatusAccepted = "accepted"
	// ChangeRequestStatusDeclined will not be merged into
	// the target history
	ChangeRequestStatusDeclined = "declined"
)

// ChangeRequest is one or more proposed additions to the history of a given dataset
type ChangeRequest struct {
	// status of this request. one of: open,accepted,declined
	Status string `json:"status"`
	// created marks the time this change request was created
	Created time.Time `json:"created"`
	// the dataset this change is aimed at. The
	// history of target must match the history
	// of change up until new history entries
	// TODO - should changes be targeting a mutable history?
	Target datastore.Key `json:"target"`
	// path to HEAD of the change history
	Path datastore.Key `json:"path"`
	// The actual change history. All relevant details must be stored
	// in the dataset itself. Title & description of the change goes
	// into this dataset's commit.
	Change *dataset.Dataset `json:"change"`
}

// AcceptChangeRequest accepts an open change request, advancing the name of the dataset
// that refer to the target path to the newly-added history
func AcceptChangeRequest(r Repo, path datastore.Key) (err error) {
	cr, err := r.GetChangeRequest(path)
	if err != nil {
		return err
	}

	cr.Status = ChangeRequestStatusAccepted
	if err := r.PutChangeRequest(path, cr); err != nil {
		return err
	}
	target := cr.Target.String()

	// TODO - place all datasets related to this history chain in the store
	ds := &dataset.Dataset{PreviousPath: path.String()}
	for {
		if ds.PreviousPath == target {
			break
		}
		// datasets can sometimes resolve over the netowork, so this get / put
		// combination is required
		ds, err = r.GetDataset(datastore.NewKey(ds.PreviousPath))
		if err != nil {
			return
		}

		if err = r.PutDataset(datastore.NewKey(ds.PreviousPath), ds); err != nil {
			return
		}
	}

	name, err := r.GetName(cr.Target)
	if err != nil {
		return err
	}

	if err := r.PutName(name, cr.Path); err != nil {
		return err
	}

	return nil
}

// DeclineChangeRequest refuses an open change request
func DeclineChangeRequest(r Repo, path datastore.Key) error {
	cr, err := r.GetChangeRequest(path)
	if err != nil {
		return err
	}
	cr.Status = ChangeRequestStatusDeclined
	return r.PutChangeRequest(path, cr)
}
