package fs_repo

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/qri/repo"
	"io/ioutil"
	"os"
)

type Namestore struct {
	basepath
}

func (n Namestore) PutName(name, path string) error {
	names, err := n.names()
	if err != nil {
		return err
	}
	names[name] = path
	return n.saveFile(names, FileNamestore)
}

func (n Namestore) GetName(name string) (string, error) {
	names, err := n.names()
	if err != nil {
		return "", err
	}
	if names[name] == "" {
		return "", repo.ErrNotFound
	}
	return names[name], nil
}

func (n Namestore) DeleteName(name string) error {
	names, err := n.names()
	if err != nil {
		return err
	}
	delete(names, name)
	return n.saveFile(names, FileNamestore)
}

func (n Namestore) Names(limit, offset int) (map[string]string, error) {
	names, err := n.names()
	if err != nil {
		return nil, err
	}

	i := 0
	added := 0
	res := map[string]string{}
	for k, v := range names {
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

func (n Namestore) NameCount() (int, error) {
	names, err := n.names()
	if err != nil {
		return 0, err
	}
	return len(names), nil
}

func (r *Namestore) names() (map[string]string, error) {
	ns := map[string]string{}
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
