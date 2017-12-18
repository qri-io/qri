package repo

import (
	"github.com/ipfs/go-datastore"
)

// MemChangeRequests is an in-memory map of change requests
type MemChangeRequests map[string]*ChangeRequest

// PutChangeRequest adds a change request to the store
func (mcr MemChangeRequests) PutChangeRequest(path datastore.Key, cr *ChangeRequest) error {
	mcr[path.String()] = cr
	return nil
}

// GetChangeRequest fetches a change request by it's path
func (mcr MemChangeRequests) GetChangeRequest(path datastore.Key) (*ChangeRequest, error) {
	if mcr[path.String()] == nil {
		return nil, datastore.ErrNotFound
	}
	return mcr[path.String()], nil
}

// DeleteChangeRequest removes a change requres from the store by it's path
func (mcr MemChangeRequests) DeleteChangeRequest(path datastore.Key) error {
	delete(mcr, path.String())
	return nil
}

// ChangeRequestsForTarget get change requests for a given ("target") dataset
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

// ListChangeRequests enumerates change requests in this store
func (mcr MemChangeRequests) ListChangeRequests(limit, offset int) ([]*ChangeRequest, error) {
	if limit == -1 && len(mcr) <= 0 {
		// default to limit of 100 entries
		limit = 100
	} else if limit == -1 {
		limit = len(mcr)
	}

	i := 0
	added := 0
	res := make([]*ChangeRequest, limit)
	for _, cr := range mcr {
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
