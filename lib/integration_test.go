package lib

import (
	"context"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/auth/key"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver"
	"github.com/qri-io/qri/remote"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestTwoActorRegistryIntegration(t *testing.T) {
	tr := NewNetworkIntegrationTestRunner(t, "integration_two_actor_registry")
	defer tr.Cleanup()

	nasim := tr.InitNasim(t)

	// - nasim creates a dataset
	ref := InitWorldBankDataset(tr.Ctx, t, nasim)

	// - nasim publishes to the registry
	PushToRegistry(t, nasim, ref.Alias())

	if err := AssertLogsEqual(nasim, tr.RegistryInst, ref); err != nil {
		t.Error(err)
	}

	p := &ListParams{}
	refs, err := NewDatasetMethods(tr.RegistryInst).ListRawRefs(tr.Ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(refs)

	hinshun := tr.InitHinshun(t)

	// - hinshun searches the registry for nasim's dataset name, gets a result
	if results := SearchFor(t, hinshun, "bank"); len(results) < 1 {
		t.Logf("expected at least one result in registry search")
	}

	// - hunshun fetches a preview of nasim's dataset
	// TODO (b5) - need to use the ref returned from search results
	t.Log(ref.String())
	Preview(t, hinshun, ref.String())

	// - hinshun pulls nasim's dataset
	Pull(tr.Ctx, t, hinshun, ref.Alias())

	if err := AssertLogsEqual(nasim, hinshun, ref); err != nil {
		t.Error(err)
	}

	// 5. nasim commits a new version
	ref = Commit2WorldBank(tr.Ctx, t, nasim)

	// 6. nasim re-publishes to the registry
	PushToRegistry(t, nasim, ref.Alias())

	// 7. hinshun logsyncs with the registry for world bank dataset, sees multiple versions
	dsm := NewDatasetMethods(hinshun)
	_, err = dsm.Pull(tr.Ctx, &PullParams{LogsOnly: true, Ref: ref.String()})
	if err != nil {
		t.Errorf("cloning logs: %s", err)
	}

	if err := AssertLogsEqual(nasim, hinshun, ref); err != nil {
		t.Error(err)
	}

	// TODO (b5) - assert hinshun DOES NOT have blocks for the latest commit to world bank dataset

	// 8. hinshun pulls latest version
	Pull(tr.Ctx, t, hinshun, ref.Alias())

	// TODO (b5) - assert hinshun has world bank dataset blocks

	// all three should now have the same HEAD reference & InitID
	dsrefspec.ConsistentResolvers(t, dsref.Ref{
		Username: ref.Username,
		Name:     ref.Name,
	},
		nasim.Repo(),
		hinshun.Repo(),
		tr.RegistryInst.Repo(),
	)
}

func TestAddCheckoutIntegration(t *testing.T) {
	tr := NewNetworkIntegrationTestRunner(t, "integration_add_checkout")
	defer tr.Cleanup()

	nasim := tr.InitNasim(t)

	// - nasim creates a dataset, publishes to registry
	ref := InitWorldBankDataset(tr.Ctx, t, nasim)
	PushToRegistry(t, nasim, ref.Alias())

	hinshun := tr.InitHinshun(t)
	dsm := NewDatasetMethods(hinshun)

	checkoutPath := filepath.Join(tr.hinshunRepo.RootPath, "wbp")
	_, err := dsm.Pull(tr.Ctx, &PullParams{
		Ref:     ref.String(),
		LinkDir: checkoutPath,
	})
	if err != nil {
		t.Errorf("adding with linked directory: %s", err)
	}

}

func TestReferencePulling(t *testing.T) {
	tr := NewNetworkIntegrationTestRunner(t, "integration_reference_pulling")
	defer tr.Cleanup()

	nasim := tr.InitNasim(t)

	// - nasim creates a dataset, publishes to registry
	ref := InitWorldBankDataset(tr.Ctx, t, nasim)
	PushToRegistry(t, nasim, ref.Alias())

	// - nasim's local repo should reflect publication
	logRes := []dsref.VersionInfo{}
	err := NewLogMethods(nasim).Log(&LogParams{Ref: ref.Alias(), ListParams: ListParams{Limit: 1}}, &logRes)
	if err != nil {
		t.Fatal(err)
	}

	if logRes[0].Published != true {
		t.Errorf("nasim has published HEAD. ref[0] published is false")
	}

	hinshun := tr.InitHinshun(t)
	sqlm := NewSQLMethods(hinshun)

	// fetch this from the registry by default
	p := &SQLQueryParams{
		Query:        "SELECT * FROM nasim/world_bank_population a LIMIT 1",
		OutputFormat: "json",
	}
	results := make([]byte, 0)
	if err := sqlm.Exec(p, &results); err != nil {
		t.Fatal(err)
	}

	// re-run. dataset should now be local, and no longer require registry to
	// resolve
	p = &SQLQueryParams{
		Query:        "SELECT * FROM nasim/world_bank_population a LIMIT 1 OFFSET 1",
		OutputFormat: "json",
		ResolverMode: "local",
	}
	results = make([]byte, 0)
	if err := sqlm.Exec(p, &results); err != nil {
		t.Fatal(err)
	}

	// create adnan
	adnan := tr.InitAdnan(t)
	dsm := NewDatasetMethods(adnan)

	// run a transform script that relies on world_bank_population, which adnan's
	// node should automatically pull to execute this script
	tfScriptData := `
wbp = load_dataset("nasim/world_bank_population")

def transform(ds, ctx):
	body = wbp.get_body() + [["g","h","i",False,3]]
	ds.set_body(body)
`
	scriptPath, err := tr.adnanRepo.WriteRootFile("transform.star", tfScriptData)
	if err != nil {
		t.Fatal(err)
	}

	saveParams := &SaveParams{
		Ref: "me/wbp_plus_one",
		FilePaths: []string{
			scriptPath,
		},
		Apply: true,
	}
	_, err = dsm.Save(tr.Ctx, saveParams)
	if err != nil {
		t.Fatal(err)
	}

	// - adnan's local repo should reflect nasim's publication
	logRes = []dsref.VersionInfo{}
	err = NewLogMethods(adnan).Log(&LogParams{Ref: ref.Alias(), ListParams: ListParams{Limit: 1}}, &logRes)
	if err != nil {
		t.Fatal(err)
	}

	if logRes[0].Published != true {
		t.Errorf("adnan's log expects head was published, ref[0] published is false")
	}
}

type NetworkIntegrationTestRunner struct {
	Ctx        context.Context
	prefix     string
	TestCrypto key.CryptoGenerator

	nasimRepo, hinshunRepo, adnanRepo *repotest.TempRepo
	Nasim, Hinshun, Adnan             *Instance

	registryRepo       *repotest.TempRepo
	Registry           registry.Registry
	RegistryInst       *Instance
	RegistryHTTPServer *httptest.Server
}

func NewNetworkIntegrationTestRunner(t *testing.T, prefix string) *NetworkIntegrationTestRunner {
	tr := &NetworkIntegrationTestRunner{
		Ctx:        context.Background(),
		prefix:     prefix,
		TestCrypto: repotest.NewTestCrypto(),
	}

	tr.InitRegistry(t)

	return tr
}

func (tr *NetworkIntegrationTestRunner) Cleanup() {
	if tr.RegistryHTTPServer != nil {
		tr.RegistryHTTPServer.Close()
	}
	if tr.registryRepo != nil {
		tr.registryRepo.Delete()
	}
	if tr.nasimRepo != nil {
		tr.nasimRepo.Delete()
	}
	if tr.hinshunRepo != nil {
		tr.hinshunRepo.Delete()
	}
}

func (tr *NetworkIntegrationTestRunner) InitNasim(t *testing.T) *Instance {
	r, err := repotest.NewTempRepo("nasim", fmt.Sprintf("%s_nasim", tr.prefix), tr.TestCrypto)
	if err != nil {
		t.Fatal(err)
	}

	if tr.RegistryHTTPServer != nil {
		cfg := r.GetConfig()
		cfg.Registry.Location = tr.RegistryHTTPServer.URL
		r.WriteConfigFile()
	}
	tr.nasimRepo = &r

	if tr.Nasim, err = NewInstance(tr.Ctx, r.QriPath, OptIOStreams(ioes.NewDiscardIOStreams())); err != nil {
		t.Fatal(err)
	}

	return tr.Nasim
}

func (tr *NetworkIntegrationTestRunner) InitHinshun(t *testing.T) *Instance {
	r, err := repotest.NewTempRepo("hinshun", fmt.Sprintf("%s_hinshun", tr.prefix), tr.TestCrypto)
	if err != nil {
		t.Fatal(err)
	}

	if tr.RegistryHTTPServer != nil {
		cfg := r.GetConfig()
		cfg.Registry.Location = tr.RegistryHTTPServer.URL
		r.WriteConfigFile()
	}
	tr.hinshunRepo = &r

	if tr.Hinshun, err = NewInstance(tr.Ctx, tr.hinshunRepo.QriPath, OptIOStreams(ioes.NewDiscardIOStreams())); err != nil {
		t.Fatal(err)
	}

	return tr.Hinshun
}

func (tr *NetworkIntegrationTestRunner) InitAdnan(t *testing.T) *Instance {
	r, err := repotest.NewTempRepo("adnan", fmt.Sprintf("%s_adnan", tr.prefix), tr.TestCrypto)
	if err != nil {
		t.Fatal(err)
	}

	if tr.RegistryHTTPServer != nil {
		cfg := r.GetConfig()
		cfg.Registry.Location = tr.RegistryHTTPServer.URL
		r.WriteConfigFile()
	}
	tr.adnanRepo = &r

	if tr.Adnan, err = NewInstance(tr.Ctx, r.QriPath, OptIOStreams(ioes.NewDiscardIOStreams())); err != nil {
		t.Fatal(err)
	}

	return tr.Adnan
}

func (tr *NetworkIntegrationTestRunner) InitRegistry(t *testing.T) {
	rr, err := repotest.NewTempRepo("registry", fmt.Sprintf("%s_registry", tr.prefix), tr.TestCrypto)
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

	tr.RegistryInst, err = NewInstance(tr.Ctx, rr.QriPath, OptIOStreams(ioes.NewDiscardIOStreams()))
	if err != nil {
		t.Fatal(err)
	}

	node := tr.RegistryInst.Node()
	if node == nil {
		t.Fatal("creating a Registry for NetworkIntegration test fails if `qri connect` is running")
	}

	rem, err := remote.NewRemote(node, cfg.Remote, node.Repo.Logbook())
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

func AssertLogsEqual(a, b *Instance, ref dsref.Ref) error {

	aLogs, err := a.logbook.DatasetRef(context.Background(), ref)
	if err != nil {
		return fmt.Errorf("fetching logs for a instance: %s", err)
	}

	bLogs, err := b.logbook.DatasetRef(context.Background(), ref)
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

func InitWorldBankDataset(ctx context.Context, t *testing.T, inst *Instance) dsref.Ref {
	res, err := NewDatasetMethods(inst).Save(ctx, &SaveParams{
		Ref: "me/world_bank_population",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "World Bank Population",
			},
			BodyPath: "body.csv",
			BodyBytes: []byte(`a,b,c,true,2
d,e,f,false,3`),
			Readme: &dataset.Readme{
				ScriptPath:  "readme.md",
				ScriptBytes: []byte("#World Bank Population\nhow many people live on this planet?"),
			},
		},
	})

	if err != nil {
		log.Fatalf("saving dataset version: %s", err)
	}

	return dsref.ConvertDatasetToVersionInfo(res).SimpleRef()
}

func Commit2WorldBank(ctx context.Context, t *testing.T, inst *Instance) dsref.Ref {
	res, err := NewDatasetMethods(inst).Save(ctx, &SaveParams{
		Ref: "me/world_bank_population",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "World Bank Population",
			},
			BodyPath: "body.csv",
			BodyBytes: []byte(`a,b,c,true,2
d,e,f,false,3
g,g,i,true,4`),
		},
	})

	if err != nil {
		log.Fatalf("saving dataset version: %s", err)
	}

	return dsref.ConvertDatasetToVersionInfo(res).SimpleRef()
}

func PushToRegistry(t *testing.T, inst *Instance, refstr string) dsref.Ref {
	res := dsref.Ref{}
	err := NewRemoteMethods(inst).Push(&PushParams{
		Ref: refstr,
	}, &res)

	if err != nil {
		t.Fatalf("publishing dataset: %s", err)
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

func Pull(ctx context.Context, t *testing.T, inst *Instance, refstr string) *dataset.Dataset {
	t.Helper()
	res, err := NewDatasetMethods(inst).Pull(ctx, &PullParams{Ref: refstr})
	if err != nil {
		t.Fatalf("cloning dataset %s: %s", refstr, err)
	}
	return res
}

func Preview(t *testing.T, inst *Instance, refstr string) *dataset.Dataset {
	t.Helper()
	p := &PreviewParams{
		Ref:        refstr,
		RemoteName: "",
	}
	res := &dataset.Dataset{}
	if err := NewRemoteMethods(inst).Preview(p, res); err != nil {
		t.Fatal(err)
	}
	return res
}
