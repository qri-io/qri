package cmd

import (
	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset/dsgraph"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/fs"
	"github.com/spf13/viper"
)

var r repo.Repo

func GetRepo() repo.Repo {
	if r != nil {
		return r
	}
	r, err := fs_repo.NewRepo(viper.GetString(QriRepoPath))
	ExitIfErr(err)
	return r
}

func LoadNamespaceGraph() map[string]datastore.Key {
	ns, err := GetRepo().Namespace()
	ExitIfErr(err)
	return ns
}

func SaveNamespaceGraph(g map[string]datastore.Key) error {
	return GetRepo().SaveNamespace(g)
}

func LoadResourceQueriesGraph() dsgraph.ResourceQueries {
	rq, err := GetRepo().ResourceQueries()
	ExitIfErr(err)
	return rq
}

func SaveResourceQueriesGraph(g dsgraph.ResourceQueries) error {
	return GetRepo().SaveResourceQueries(g)
}

func LoadQueryResultsGraph() dsgraph.QueryResults {
	g, err := GetRepo().QueryResults()
	ExitIfErr(err)
	return g
}

func SaveQueryResultsGraph(g dsgraph.QueryResults) error {
	return GetRepo().SaveQueryResults(g)
}
