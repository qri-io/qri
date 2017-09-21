package repo

// Namespace is an in-progress solution for aliasing
// datasets locally
type Namestore interface {
	PutName(name, path string) error
	GetName(name string) (string, error)
	DeleteName(name string) error
	Names(limit, offset int) (map[string]string, error)
	NameCount() (int, error)
}

// MemNamestore is an in-memory implementation of the Namestore interface
type MemNamestore map[string]string

func (r MemNamestore) PutName(name, path string) error {
	r[name] = path
	return nil
}

func (r MemNamestore) GetName(name string) (string, error) {
	if r[name] == "" {
		return "", ErrNotFound
	}
	return r[name], nil
}

func (r MemNamestore) DeleteName(name string) error {
	delete(r, name)
	return nil
}

func (r MemNamestore) Names(limit, offset int) (map[string]string, error) {
	i := 0
	added := 0
	res := map[string]string{}
	for k, v := range r {
		if i < offset {
			continue
		}

		if limit > 0 && added < limit {
			res[k] = v
			added++
		} else if added == limit {
			break
		}

		i++
	}
	return res, nil
}

func (r MemNamestore) NameCount() (int, error) {
	return len(r), nil
}
