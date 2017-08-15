package repo

// import (
// 	"github.com/ipfs/go-datastore"
// )

// MapRepo is a repository implementation that uses an in-memory
// map as it's datastore.
// type MapRepo struct {
// 	store datastore.Datastore
// }

// // Store gives the Datastore Repo's backing store
// func (r *MapRepo) Store() datastore.Datastore {
// 	return r.store
// }

// // NewMapRepo returns a repository with an in-memory datastore
// func NewMapRepo() (Repo, error) {
// 	return &MapRepo{
// 		store: datastore.NewMapDatastore(),
// 	}, nil
// }

// func (r *MapRepo) Namespace() (map[string]datastore.Key, error) {
// 	g := map[string]datastore.Key{}
// 	v, err := r.store.Get(RepoFileKey(RfNamespace))
// 	if err != nil {
// 		if err == datastore.ErrNotFound {
// 			// if no key exists, return an empty namespace
// 			return g, nil
// 		}
// 		return nil, fmt.Errorf("error loading namespace graph:", err.Error())
// 	}

// 	return g, nil
// }

// func SaveNamespace(path string, graph map[string]datastore.Key) error {
// 	data, err := json.Marshal(graph)
// 	if err != nil {
// 		return err
// 	}

// 	return ioutil.WriteFile(path, data, os.ModePerm)
// }

// func LoadQueryResults(path string) dsgraph.QueryResults {
// 	r := dsgraph.QueryResults{}
// 	data, err := ioutil.ReadFile(path)
// 	if err != nil {
// 		fmt.Println("error loading query results graph:", err.Error())
// 		return r
// 	}

// 	if err := json.Unmarshal(data, &r); err != nil {
// 		fmt.Println("error unmarshaling query results graph:", err.Error())
// 		return dsgraph.QueryResults{}
// 	}
// 	return r
// }

// func SaveQueryResults(path string, graph dsgraph.QueryResults) error {
// 	data, err := json.Marshal(graph)
// 	if err != nil {
// 		return err
// 	}

// 	return ioutil.WriteFile(path, data, os.ModePerm)
// }

// func LoadResourceQueries(path string) dsgraph.ResourceQueries {
// 	r := dsgraph.ResourceQueries{}
// 	data, err := ioutil.ReadFile(path)
// 	if err != nil {
// 		fmt.Println("error loading resource queries graph:", err.Error())
// 		return r
// 	}

// 	if err := json.Unmarshal(data, &r); err != nil {
// 		fmt.Println("error unmarshaling resource queries graph:", err.Error())
// 		return dsgraph.ResourceQueries{}
// 	}
// 	return r
// }

// func SaveResourceQueries(path string, graph dsgraph.ResourceQueries) error {
// 	data, err := json.Marshal(graph)
// 	if err != nil {
// 		return err
// 	}

// 	return ioutil.WriteFile(path, data, os.ModePerm)
// }

// func LoadResourceMeta(path string) dsgraph.ResourceMeta {
// 	r := dsgraph.ResourceMeta{}
// 	data, err := ioutil.ReadFile(path)
// 	if err != nil {
// 		fmt.Println("error loading resource meta graph:", err.Error())
// 		return r
// 	}

// 	if err := json.Unmarshal(data, &r); err != nil {
// 		fmt.Println("error unmarshaling resource meta graph:", err.Error())
// 		return dsgraph.ResourceMeta{}
// 	}
// 	return r
// }

// func SaveResourceMeta(path string, graph dsgraph.ResourceMeta) error {
// 	data, err := json.Marshal(graph)
// 	if err != nil {
// 		return err
// 	}

// 	return ioutil.WriteFile(path, data, os.ModePerm)
// }

// func ()
