package repo

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ds-flatfs"
)

// Repo is a repository implementation that uses an ipfs datastore as
// it's backing store
type Repo struct {
	store datastore.Datastore
}

// Store gives the Datastore Repo's backing store
func (r *Repo) Store() datastore.Datastore {
	return r.store
}

// NewMapRepo returns a repository with an in-memory datastore
func NewMapRepo() (Repo, error) {
	return &Repo{
		store: datastore.NewMapDatastore(),
	}, nil
}

// NewFlatFsRepo creates a flat filestore repository
func NewFlatFsRepo(basepath string) (Repo, error) {
	ffs, err := flatfs.CreateOrOpen(basepath, flatfs.Prefix(2), false)
	if err != nil {
		return nil, err
	}

	return &Repo{
		store: ffs,
	}, nil
}

func (r *Repo) NamespaceGraph() (map[string]datastore.Key, error) {
	ns := map[string]datastore.Key{}
	v, err := r.store.Get(RepoFileKey(RfNamespace))
	if err != nil {
		if err == datastore.ErrNotFound {
			// if no key exists, return an empty namespace
			return ns, nil
		}
		return nil, fmt.Errorf("error loading namespace graph:", err.Error())
	}

	switch data := v.(type) {
	case []byte:
		if err := json.Unmarshal(data, &ns); err != nil {
			return ns, fmt.Errorf("error unmarshaling namespace graph:", err.Error())
		}
		return ns, nil
	case map[string]datastore.Key:
		return data, nil
	default:
		return ns, fmt.Errorf("invalid storage type for namespace")
	}
}

func SaveNamespaceGraph(path string, graph map[string]datastore.Key) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, os.ModePerm)
}

func LoadQueryResultsGraph(path string) dsgraph.QueryResults {
	r := dsgraph.QueryResults{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error loading query results graph:", err.Error())
		return r
	}

	if err := json.Unmarshal(data, &r); err != nil {
		fmt.Println("error unmarshaling query results graph:", err.Error())
		return dsgraph.QueryResults{}
	}
	return r
}

func SaveQueryResultsGraph(path string, graph dsgraph.QueryResults) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, os.ModePerm)
}

func LoadResourceQueriesGraph(path string) dsgraph.ResourceQueries {
	r := dsgraph.ResourceQueries{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error loading resource queries graph:", err.Error())
		return r
	}

	if err := json.Unmarshal(data, &r); err != nil {
		fmt.Println("error unmarshaling resource queries graph:", err.Error())
		return dsgraph.ResourceQueries{}
	}
	return r
}

func SaveResourceQueriesGraph(path string, graph dsgraph.ResourceQueries) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, os.ModePerm)
}

func LoadResourceMetaGraph(path string) dsgraph.ResourceMeta {
	r := dsgraph.ResourceMeta{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error loading resource meta graph:", err.Error())
		return r
	}

	if err := json.Unmarshal(data, &r); err != nil {
		fmt.Println("error unmarshaling resource meta graph:", err.Error())
		return dsgraph.ResourceMeta{}
	}
	return r
}

func SaveResourceMetaGraph(path string, graph dsgraph.ResourceMeta) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, os.ModePerm)
}
