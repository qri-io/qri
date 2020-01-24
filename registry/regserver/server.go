// Package regserver is a wrapper around the handlers package,
// turning it into a proper http server
package main

import (
	"context"
	"net/http"
	"os"

	logger "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver/handlers"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

var (
	log      = logger.Logger("regserver")
	adminKey string
)

func main() {
	logger.SetLogLevel("regserver", "info")
	ctx := context.Background()
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mock, err := repotest.NewMockRepo("registry", "qri_registry")
	if err != nil {
		log.Fatalf("creating temporary repo: %s", err)
	}
	defer mock.Delete()

	cfg := mock.GetConfig()
	cfg.Registry.Location = ""
	cfg.Remote = &config.Remote{
		Enabled:          true,
		AcceptSizeMax:    -1,
		AcceptTimeoutMs:  -1,
		RequireAllBlocks: false,
		AllowRemoves:     true,
	}

	mock.WriteConfigFile()

	opts := []lib.Option{
		lib.OptSetIPFSPath(mock.IPFSPath),
	}

	inst, err := lib.NewInstance(ctx, mock.QriPath, opts...)
	if err != nil {
		log.Fatalf("creating qri instance: %s", err)
	}

	addBasicDataset(inst)

	rem, err := remote.NewRemote(inst.Node(), cfg.Remote)
	if err != nil {
		log.Panic("creating remote: %s", err)
	}

	mux := http.NewServeMux()

	mux.Handle("/remote/dsync", rem.DsyncHTTPHandler())
	mux.Handle("/remote/refs", rem.RefsHTTPHandler()) // deprecate this
	mux.Handle("/remote/logsync", rem.LogsyncHTTPHandler())
	mux.Handle("/remote/feeds/", rem.FeedsHTTPHandler())
	mux.Handle("/remote/preview/", rem.PreviewHTTPHandler())
	mux.Handle("/remote/component/", rem.ComponentHTTPHandler())

	ps := registry.NewMemProfiles()
	mux.HandleFunc("/registry/profile", handlers.NewProfileHandler(ps))
	mux.HandleFunc("/registry/profiles", handlers.NewProfilesHandler(ps))
	mux.HandleFunc("/registry/search", handlers.NewSearchHandler(searchShim{inst.Repo()}))

	s := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Infof("serving on: %s", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Info(err.Error())
	}
}

func addBasicDataset(inst *lib.Instance) {
	dsm := lib.NewDatasetRequestsInstance(inst)
	res := repo.DatasetRef{}
	err := dsm.Save(&lib.SaveParams{
		Publish: true,
		Ref:     "me/dataset",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "I'm a dataset",
			},
			BodyPath: "body.csv",
			BodyBytes: []byte(`a,b,c,true,2
d,e,f,false,3`),
		},
	}, &res)

	if err != nil {
		log.Fatalf("saving dataset verion: %s", err)
	}
}

type searchShim struct {
	repo.Repo
}

func (ss searchShim) Search(p registry.SearchParams) ([]*dataset.Dataset, error) {
	ctx := context.Background()
	refs, err := base.ListDatasets(ctx, ss.Repo, p.Q, 1000, 0, false, true, false)
	if err != nil {
		return nil, err
	}

	var res []*dataset.Dataset
	for _, ref := range refs {
		res = append(res, ref.Dataset)
	}
	return res, nil
}
