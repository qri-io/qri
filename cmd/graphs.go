package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
)

func LoadQueryResultsGraph() dsgraph.QueryResults {
	r := dsgraph.QueryResults{}
	data, err := ioutil.ReadFile(viper.GetString(QueryResultsGraphPath))
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

func SaveQueryResultsGraph(graph dsgraph.QueryResults) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(viper.GetString(QueryResultsGraphPath), data, os.ModePerm)
}

func LoadResourceQueriesGraph() dsgraph.ResourceQueries {
	r := dsgraph.ResourceQueries{}
	data, err := ioutil.ReadFile(viper.GetString(ResourceQueriesGraphPath))
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

func SaveResourceQueriesGraph(graph dsgraph.ResourceQueries) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(viper.GetString(ResourceQueriesGraphPath), data, os.ModePerm)
}

func LoadResourceMetaGraph() dsgraph.ResourceMeta {
	r := dsgraph.ResourceMeta{}
	data, err := ioutil.ReadFile(viper.GetString(ResourceMetaGraphPath))
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

func SaveResourceMetaGraph(graph dsgraph.ResourceMeta) error {
	data, err := json.Marshal(graph)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(viper.GetString(ResourceMetaGraphPath), data, os.ModePerm)
}
