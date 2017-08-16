package fs_repo

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/peer_repo"
	"github.com/qri-io/qri/repo/profile"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Repo struct {
	base string
}

func NewRepo(base string) (repo.Repo, error) {
	if err := os.MkdirAll(base, os.ModePerm); err != nil {
		return nil, err
	}
	return &Repo{
		base: base,
	}, nil
}

func (r *Repo) filepath(rf repo.File) string {
	return filepath.Join(r.base, fmt.Sprintf("%s.json", repo.Filepath(rf)))
}

func (r *Repo) Profile() (*profile.Profile, error) {
	p := &profile.Profile{}
	data, err := ioutil.ReadFile(r.filepath(repo.FileProfile))
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return p, fmt.Errorf("error loading profile: %s", err.Error())
	}

	if err := json.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("error unmarshaling profile: %s", err.Error())
	}

	return p, nil
}

func (r *Repo) SaveProfile(p *profile.Profile) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(r.filepath(repo.FileProfile), data, os.ModePerm)
}

func (r *Repo) Namespace() (map[string]datastore.Key, error) {
	g := map[string]datastore.Key{}
	data, err := ioutil.ReadFile(r.filepath(repo.FileNamespace))
	if err != nil {
		if os.IsNotExist(err) {
			return g, nil
		}
		return g, fmt.Errorf("error loading namespace graph:", err.Error())
	}

	if err := json.Unmarshal(data, &g); err != nil {
		return g, fmt.Errorf("error unmarshaling namespace graph:", err.Error())
	}

	return g, nil
}

func (r *Repo) SaveNamespace(graph map[string]datastore.Key) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(r.filepath(repo.FileNamespace), data, os.ModePerm)
}

func (r *Repo) QueryResults() (dsgraph.QueryResults, error) {
	g := dsgraph.QueryResults{}
	data, err := ioutil.ReadFile(r.filepath(repo.FileQueryResults))
	if err != nil {
		if os.IsNotExist(err) {
			return g, nil
		}
		return g, fmt.Errorf("error loading query results graph:", err.Error())
	}

	if err := json.Unmarshal(data, &g); err != nil {
		return g, fmt.Errorf("error unmarshaling query results graph:", err.Error())
	}
	return g, nil
}

func (r *Repo) SaveQueryResults(graph dsgraph.QueryResults) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(r.filepath(repo.FileQueryResults), data, os.ModePerm)
}

func (r *Repo) ResourceQueries() (dsgraph.ResourceQueries, error) {
	g := dsgraph.ResourceQueries{}
	data, err := ioutil.ReadFile(r.filepath(repo.FileResourceQueries))
	if err != nil {
		if os.IsNotExist(err) {
			return g, nil
		}
		return g, fmt.Errorf("error loading resource queries graph:", err.Error())
	}

	if err := json.Unmarshal(data, &g); err != nil {
		return g, fmt.Errorf("error unmarshaling resource queries graph:", err.Error())
	}
	return g, nil
}

func (r *Repo) SaveResourceQueries(graph dsgraph.ResourceQueries) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(r.filepath(repo.FileResourceQueries), data, os.ModePerm)
}

func (r *Repo) ResourceMeta() (dsgraph.ResourceMeta, error) {
	g := dsgraph.ResourceMeta{}
	data, err := ioutil.ReadFile(r.filepath(repo.FileResourceMeta))
	if err != nil {
		if os.IsNotExist(err) {
			return g, nil
		}
		return g, fmt.Errorf("error loading resource meta graph:", err.Error())
	}

	if err := json.Unmarshal(data, &g); err != nil {
		return g, fmt.Errorf("error unmarshaling resource meta graph:", err.Error())
	}

	return g, nil
}

func (r *Repo) SaveResourceMeta(graph dsgraph.ResourceMeta) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(r.filepath(repo.FileResourceMeta), data, os.ModePerm)
}

func (r *Repo) Peers() (map[string]*peer_repo.Repo, error) {
	p := map[string]*peer_repo.Repo{}
	data, err := ioutil.ReadFile(r.filepath(repo.FilePeerRepos))
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return p, fmt.Errorf("error loading peers: %s", err.Error())
	}

	if err := json.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("error unmarshaling peers: %s", err.Error())
	}

	return p, nil
}

func (r *Repo) SavePeers(p map[string]*peer_repo.Repo) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(r.filepath(repo.FilePeerRepos), data, os.ModePerm)
}

func (r *Repo) Destroy() error {
	return os.RemoveAll(r.base)
}
