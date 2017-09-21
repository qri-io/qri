package fs_repo

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/repo"
	"io/ioutil"
	"os"
)

type Namestore struct {
	basepath
}

func (n Namestore) PutName(name string, path datastore.Key) error {
	names, err := n.names()
	if err != nil {
		return err
	}
	names[name] = path
	return n.saveFile(names, FileNamestore)
}

func (n Namestore) GetPath(name string) (datastore.Key, error) {
	names, err := n.names()
	if err != nil {
		return datastore.NewKey(""), err
	}
	if names[name].String() == "" {
		return datastore.NewKey(""), repo.ErrNotFound
	}
	return names[name], nil
}

func (n Namestore) GetName(path datastore.Key) (string, error) {
	names, err := n.names()
	if err != nil {
		return "", err
	}
	for name, p := range names {
		if path.Equal(p) {
			return name, nil
		}
	}
	return "", repo.ErrNotFound
}

func (n Namestore) DeleteName(name string) error {
	names, err := n.names()
	if err != nil {
		return err
	}
	delete(names, name)
	return n.saveFile(names, FileNamestore)
}

func (n Namestore) Namespace(limit, offset int) (map[string]datastore.Key, error) {
	names, err := n.names()
	if err != nil {
		return nil, err
	}

	i := 0
	added := 0
	res := map[string]datastore.Key{}
	for name, path := range names {
		if i < offset {
			continue
		}

		if limit > 0 && added < limit {
			res[name] = path
			added++
		} else if added == limit {
			break
		}

		i++
	}
	return res, nil
}

func (n Namestore) NameCount() (int, error) {
	names, err := n.names()
	if err != nil {
		return 0, err
	}
	return len(names), nil
}

func (r *Namestore) names() (map[string]datastore.Key, error) {
	ns := map[string]datastore.Key{}
	data, err := ioutil.ReadFile(r.filepath(FileNamestore))
	if err != nil {
		if os.IsNotExist(err) {
			return ns, nil
		}
		return ns, fmt.Errorf("error loading names: %s", err.Error())
	}

	if err := json.Unmarshal(data, &ns); err != nil {
		return ns, fmt.Errorf("error unmarshaling names: %s", err.Error())
	}
	return ns, nil
}
