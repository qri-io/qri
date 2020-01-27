package main

import (
	"context"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
	repotest "github.com/qri-io/qri/repo/test"
)

// NewTempRepoRegistry creates a temporary repo & builds a registry atop it.
// callers should always call the returned cleanup function when finished to
// remove temp files
func NewTempRepoRegistry(ctx context.Context) (*lib.Instance, registry.Registry, func(), error) {
	mock, err := repotest.NewTempRepo("registry", "qri_registry")
	if err != nil {
		return nil, registry.Registry{}, nil, err
	}
	cleanup := mock.Delete

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
		return nil, registry.Registry{}, nil, err
	}

	rem, err := remote.NewRemote(inst.Node(), cfg.Remote)
	if err != nil {
		return nil, registry.Registry{}, nil, err
	}

	reg := registry.Registry{
		Remote:   rem,
		Profiles: registry.NewMemProfiles(),
		Search:   regserver.MockRepoSearch{Repo: inst.Repo()},
	}

	return inst, reg, cleanup, nil
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
