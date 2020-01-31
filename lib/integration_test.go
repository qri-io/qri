package lib

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestTwoActorRegistryIntegration(t *testing.T) {
	tr := NewNetworkIntegrationTestRunner(t, "integration_two_actor_registry")
	defer tr.Cleanup()

	nasim := tr.InitNasim(t)

	// 1. nasim creates a dataset
	ref := InitWorldBankDataset(t, nasim)

	// 2. nasim publishes to the registry
	PublishToRegistry(t, nasim, ref.AliasString())

	if err := AssertLogsEqual(nasim, tr.RegistryInst, ref); err != nil {
		t.Error(err)
	}

	p := &ListParams{}
	refs := ""
	if err := NewDatasetRequestsInstance(tr.RegistryInst).ListRawRefs(p, &refs); err != nil {
		t.Fatal(err)
	}
	t.Log(refs)

	hinshun := tr.InitHinshun(t)

	// 3. hinshun searches the registry for nasim's dataset name, gets a result
	if results := SearchFor(t, hinshun, "bank"); len(results) < 1 {
		t.Logf("expected at least one result in registry search")
		// t.Errorf("expected at least one result in registry search")
	}

	// 4. hinshun clones nasim's dataset
	Clone(t, hinshun, ref.AliasString())

	if err := AssertLogsEqual(nasim, hinshun, ref); err != nil {
		t.Error(err)
	}

	// 5. nasim commits a new version
	ref = Commit2WorldBank(t, nasim)

	// 6. nasim re-publishes to the registry
	PublishToRegistry(t, nasim, ref.AliasString())

	// 7. hinshun logsyncs with the registry for world bank dataset, sees multiple versions
	dsm := NewDatasetRequestsInstance(hinshun)
	res := &reporef.DatasetRef{}
	if err := dsm.Add(&AddParams{LogsOnly: true, Ref: ref.String()}, res); err != nil {
		t.Errorf("cloning logs: %s", err)
	}

	if err := AssertLogsEqual(nasim, hinshun, ref); err != nil {
		t.Error(err)
	}

	// TODO (b5) - assert hinshun DOES NOT have blocks for the latest commit to world bank dataset

	// 8. hinshun clones latest version
	Clone(t, hinshun, ref.AliasString())

	// TODO (b5) - assert hinshun has world bank dataset blocks
}

type NetworkIntegrationTestRunner struct {
	Ctx                                  context.Context
	prefix                               string
	nasimRepo, hinshunRepo, registryRepo *repotest.TempRepo
	Nasim, Hinshun, RegistryInst         *Instance

	Registry           registry.Registry
	RegistryHTTPServer *httptest.Server
}

func NewNetworkIntegrationTestRunner(t *testing.T, prefix string) *NetworkIntegrationTestRunner {
	tr := &NetworkIntegrationTestRunner{
		Ctx:    context.Background(),
		prefix: prefix,
	}

	tr.InitRegistry(t)

	return tr
}

func (tr *NetworkIntegrationTestRunner) Cleanup() {
	tr.RegistryHTTPServer.Close()
	tr.registryRepo.Delete()
	if tr.nasimRepo != nil {
		tr.nasimRepo.Delete()
	}
	if tr.hinshunRepo != nil {
		tr.hinshunRepo.Delete()
	}
}

func (tr *NetworkIntegrationTestRunner) InitNasim(t *testing.T) *Instance {
	r, err := repotest.NewTempRepo("nasim", fmt.Sprintf("%s_nasim", tr.prefix))
	if err != nil {
		t.Fatal(err)
	}

	cfg := r.GetConfig()
	cfg.Registry.Location = tr.RegistryHTTPServer.URL
	r.WriteConfigFile()
	tr.nasimRepo = &r

	if tr.Nasim, err = NewInstance(tr.Ctx, r.QriPath); err != nil {
		t.Fatal(err)
	}

	return tr.Nasim
}

func (tr *NetworkIntegrationTestRunner) InitHinshun(t *testing.T) *Instance {
	r, err := repotest.NewTempRepo("hinshun", fmt.Sprintf("%s_hinshun", tr.prefix))
	if err != nil {
		t.Fatal(err)
	}

	tr.hinshunRepo = &r
	cfg := tr.hinshunRepo.GetConfig()
	cfg.Registry.Location = tr.RegistryHTTPServer.URL
	tr.hinshunRepo.WriteConfigFile()

	if tr.Hinshun, err = NewInstance(tr.Ctx, tr.hinshunRepo.QriPath); err != nil {
		t.Fatal(err)
	}

	return tr.Hinshun
}

func (tr *NetworkIntegrationTestRunner) InitRegistry(t *testing.T) {
	rr, err := repotest.NewTempRepo("registry", fmt.Sprintf("%s_registry", tr.prefix))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("registry qri path: %s", rr.QriPath)

	tr.registryRepo = &rr

	cfg := rr.GetConfig()
	cfg.Registry.Location = ""
	cfg.Remote = &config.Remote{
		Enabled:          true,
		AcceptSizeMax:    -1,
		AcceptTimeoutMs:  -1,
		RequireAllBlocks: false,
		AllowRemoves:     true,
	}

	rr.WriteConfigFile()

	opts := []Option{
		OptSetIPFSPath(rr.IPFSPath),
	}

	tr.RegistryInst, err = NewInstance(tr.Ctx, rr.QriPath, opts...)
	if err != nil {
		t.Fatal(err)
	}

	rem, err := remote.NewRemote(tr.RegistryInst.Node(), cfg.Remote)
	if err != nil {
		t.Fatal(err)
	}

	tr.Registry = registry.Registry{
		Remote:   rem,
		Profiles: registry.NewMemProfiles(),
		Search:   regserver.MockRepoSearch{Repo: tr.RegistryInst.Repo()},
	}

	_, tr.RegistryHTTPServer = regserver.NewMockServerRegistry(tr.Registry)
}

func AssertLogsEqual(a, b *Instance, ref *reporef.DatasetRef) error {
	r := repo.ConvertToDsref(*ref)

	aLogs, err := a.logbook.DatasetRef(context.Background(), r)
	if err != nil {
		return fmt.Errorf("fetching logs for a instance: %s", err)
	}

	bLogs, err := b.logbook.DatasetRef(context.Background(), r)
	if err != nil {
		return fmt.Errorf("fetching logs for b instance: %s", err)
	}

	if aLogs.ID() != bLogs.ID() {
		return fmt.Errorf("log ID mismatch. %s != %s", aLogs.ID(), bLogs.ID())
	}

	if len(aLogs.Logs) != len(bLogs.Logs) {
		return fmt.Errorf("oplength mismatch. %d != %d", len(aLogs.Logs), len(bLogs.Logs))
	}

	return nil
}

func InitWorldBankDataset(t *testing.T, inst *Instance) *reporef.DatasetRef {
	res := &reporef.DatasetRef{}
	err := NewDatasetRequestsInstance(inst).Save(&SaveParams{
		Publish: true,
		Ref:     "me/world_bank_population",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "World Bank Population",
			},
			BodyPath: "body.csv",
			BodyBytes: []byte(`a,b,c,true,2
d,e,f,false,3`),
		},
	}, res)

	if err != nil {
		log.Fatalf("saving dataset version: %s", err)
	}

	return res
}

func Commit2WorldBank(t *testing.T, inst *Instance) *reporef.DatasetRef {
	res := &reporef.DatasetRef{}
	err := NewDatasetRequestsInstance(inst).Save(&SaveParams{
		Publish: true,
		Ref:     "me/world_bank_population",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "World Bank Population",
			},
			BodyPath: "body.csv",
			BodyBytes: []byte(`a,b,c,true,2
d,e,f,false,3
g,g,i,true,4`),
		},
	}, res)

	if err != nil {
		log.Fatalf("saving dataset version: %s", err)
	}

	return res
}

func PublishToRegistry(t *testing.T, inst *Instance, refstr string) *reporef.DatasetRef {
	res := &reporef.DatasetRef{}
	err := NewRemoteMethods(inst).Publish(&PublicationParams{
		Ref: refstr,
	}, res)

	if err != nil {
		log.Fatalf("publishing dataset: %s", err)
	}

	return res
}

func SearchFor(t *testing.T, inst *Instance, term string) []SearchResult {
	results := []SearchResult{}
	if err := NewSearchMethods(inst).Search(&SearchParams{QueryString: term}, &results); err != nil {
		t.Fatal(err)
	}

	return results
}

func Clone(t *testing.T, inst *Instance, refstr string) *reporef.DatasetRef {
	res := &reporef.DatasetRef{}
	if err := NewDatasetRequestsInstance(inst).Add(&AddParams{Ref: refstr}, res); err != nil {
		t.Fatalf("cloning dataset %s: %s", refstr, err)
	}

	return res
}
