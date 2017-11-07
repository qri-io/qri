package fs_repo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo"
)

type ChangeRequests struct {
	basepath
	file File
}

func NewChangeRequests(base string, file File) ChangeRequests {
	return ChangeRequests{basepath: basepath(base), file: file}
}

// Put a change request into the store
func (r ChangeRequests) PutChangeRequest(path datastore.Key, cr *repo.ChangeRequest) error {
	crs, err := r.changeRequests()
	if err != nil {
		return err
	}
	crs[path.String()] = cr
	return r.saveFile(cr, r.file)
}

func (r ChangeRequests) DeleteChangeRequest(path datastore.Key) error {
	cr, err := r.changeRequests()
	if err != nil {
		return err
	}
	delete(cr, path.String())
	return r.saveFile(cr, r.file)
}

// Get a change request by it's path
func (r ChangeRequests) GetChangeRequest(path datastore.Key) (*repo.ChangeRequest, error) {
	crs, err := r.changeRequests()
	if err != nil {
		return nil, err
	}

	cr := crs[path.String()]
	if cr == nil {
		return nil, datastore.ErrNotFound
	}
	return cr, nil
}

// get change requests for a given target
func (r ChangeRequests) ChangeRequestsForTarget(target datastore.Key, limit, offset int) ([]*repo.ChangeRequest, error) {
	crs, err := r.changeRequests()
	if err != nil {
		return nil, err
	}

	results := []*repo.ChangeRequest{}
	skipped := 0
	for _, cr := range crs {
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
func (r ChangeRequests) ListChangeRequests(limit, offset int) ([]*repo.ChangeRequest, error) {
	crs, err := r.changeRequests()
	if err != nil {
		return nil, err
	}

	if limit == -1 && len(crs) <= 0 {
		// default to limit of 100 entries
		limit = 100
	} else if limit == -1 {
		limit = len(crs)
	}

	i := 0
	added := 0
	res := make([]*repo.ChangeRequest, limit)
	for _, cr := range crs {
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

func (r ChangeRequests) changeRequests() (map[string]*repo.ChangeRequest, error) {
	ds := map[string]*repo.ChangeRequest{}
	data, err := ioutil.ReadFile(r.filepath(r.file))
	if err != nil {
		if os.IsNotExist(err) {
			return ds, nil
		}
		return ds, fmt.Errorf("error loading changeRequests: %s", err.Error())
	}

	if err := json.Unmarshal(data, &ds); err != nil {
		return ds, fmt.Errorf("error unmarshaling changeRequests: %s", err.Error())
	}
	return ds, nil
}
