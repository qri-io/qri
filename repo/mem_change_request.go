package repo

import (
	"github.com/ipfs/go-datastore"
)

type MemChangeRequests map[string]*ChangeRequest

// Put a change request into the store
func (mcr MemChangeRequests) PutChangeRequest(path datastore.Key, cr *ChangeRequest) error {
	mcr[path.String()] = cr
	return nil
}

// Get a change request by it's path
func (mcr MemChangeRequests) GetChangeRequest(path datastore.Key) (*ChangeRequest, error) {
	if mcr[path.String()] == nil {
		return nil, datastore.ErrNotFound
	}
	return mcr[path.String()], nil
}

func (mcr MemChangeRequests) DeleteChangeRequest(path datastore.Key) error {
	delete(mcr, path.String())
	return nil
}

// get change requests for a given target
func (mcr MemChangeRequests) ChangeRequestsForTarget(target datastore.Key, limit, offset int) ([]*ChangeRequest, error) {
	results := []*ChangeRequest{}
	skipped := 0
	for _, cr := range mcr {
		if cr.Target == target {
			if skipped < offset {
				skipped++
				continue
			}
			results = append(results, cr)
		}
		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

// list change requests in this store
func (r MemChangeRequests) ListChangeRequests(limit, offset int) ([]*ChangeRequest, error) {
	if limit == -1 && len(r) <= 0 {
		// default to limit of 100 entries
		limit = 100
	} else if limit == -1 {
		limit = len(r)
	}

	i := 0
	added := 0
	res := make([]*ChangeRequest, limit)
	for _, cr := range r {
		if i < offset {
			continue
		}

		if limit > 0 && added < limit {
			res[i] = cr
			added++
		} else if added == limit {
			break
		}

		i++
	}
	return res[:added], nil
}
