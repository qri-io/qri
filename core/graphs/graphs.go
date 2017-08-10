package graphs

import (
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsgraph"
	"io/ioutil"
	"os"
)

func LoadNamespaceGraph(path string) map[string]datastore.Key {
	r := map[string]datastore.Key{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error loading query results graph:", err.Error())
		return r
	}

	if err := json.Unmarshal(data, &r); err != nil {
		fmt.Println("error unmarshaling query results graph:", err.Error())
		return map[string]datastore.Key{}
	}
	return r
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
