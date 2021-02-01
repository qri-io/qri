package test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"runtime"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// base64-encoded Test Private Key, decoded in init
// peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
var (
	_testPk   = []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)
	testPk    = []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)
	privKey   crypto.PrivKey
	profileID = "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"

	testPeerProfile = &profile.Profile{
		Peername: "peer",
		ID:       profile.IDB58MustDecode(profileID),
	}
)

func init() {
	data, err := base64.StdEncoding.DecodeString(string(testPk))
	if err != nil {
		panic(err)
	}
	testPk = data

	privKey, err = crypto.UnmarshalPrivateKey(testPk)
	if err != nil {
		panic(fmt.Errorf("error unmarshaling private key: %s", err.Error()))
	}
	testPeerProfile.PrivKey = privKey
}

// TestdataPath returns the absolute path to a file in the testdata diretory
func TestdataPath(path string) string {
	// Get the testdata directory relative to this source file.
	_, currfile, _, _ := runtime.Caller(0)
	testdataPath := filepath.Join(filepath.Dir(currfile), "testdata")
	return filepath.Join(testdataPath, path)
}

// ProfileConfig returns the test profile as a config.Profile
func ProfileConfig() *config.ProfilePod {
	return &config.ProfilePod{
		Peername: "peer",
		ID:       profileID,
		PrivKey:  string(_testPk),
		Type:     "peer",
	}
}

// NewEmptyTestRepo initializes a test repo with no contents
func NewEmptyTestRepo(bus event.Bus) (mr *repo.MemRepo, err error) {
	ctx := context.TODO()
	pro := &profile.Profile{
		Peername: "peer",
		ID:       profile.IDB58MustDecode(profileID),
		PrivKey:  privKey,
	}
	return repo.NewMemRepoWithProfile(ctx, pro, newTestFS(ctx), bus)
}

func newTestFS(ctx context.Context) *muxfs.Mux {
	fs, err := muxfs.New(ctx, []qfs.Config{
		{Type: "mem"},
		{Type: "local"},
		{Type: "http"},
	})
	if err != nil {
		panic(err)
	}
	return fs
}

// NewTestRepo generates a repository usable for testing purposes
func NewTestRepo() (mr *repo.MemRepo, err error) {
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	mr, err = NewEmptyTestRepo(event.NilBus)
	if err != nil {
		return
	}

	for _, dsDirName := range datasets {
		tc, err := dstest.NewTestCaseFromDir(TestdataPath(dsDirName))
		if err != nil {
			return nil, err
		}
		if _, err := createDataset(mr, tc); err != nil {
			return nil, fmt.Errorf("%s error creating dataset: %s", dsDirName, err.Error())
		}
	}

	return
}

// NewTestRepoWithHistory generates a repository with a dataset that has a history, usable for testing purposes
func NewTestRepoWithHistory() (mr *repo.MemRepo, refs []reporef.DatasetRef, err error) {
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	mr, err = NewEmptyTestRepo(event.NilBus)
	if err != nil {
		return
	}

	prevPath := ""
	for _, dsDirName := range datasets {
		tc, err := dstest.NewTestCaseFromDir(TestdataPath(dsDirName))
		if err != nil {
			return nil, nil, err
		}
		tc.Input.Name = "logtest"
		tc.Input.PreviousPath = prevPath
		ref, err := createDataset(mr, tc)
		if err != nil {
			return nil, nil, fmt.Errorf("%s error creating dataset: %s", dsDirName, err.Error())
		}
		prevPath = ref.Path
		refs = append(refs, ref)
	}

	// return refs with the first ref as the head of the log
	for i := len(refs)/2 - 1; i >= 0; i-- {
		opp := len(refs) - 1 - i
		refs[i], refs[opp] = refs[opp], refs[i]
	}

	return
}

// NewTestRepoFromProfileID constructs a repo from a profileID, usable for tests
func NewTestRepoFromProfileID(id profile.ID, peerNum int, dataIndex int) (repo.Repo, error) {
	ctx := context.TODO()
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	pk, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	r, err := repo.NewMemRepoWithProfile(ctx, &profile.Profile{
		ID:       id,
		Peername: fmt.Sprintf("test-repo-%d", peerNum),
		PrivKey:  pk,
	}, newTestFS(ctx), event.NilBus)
	if err != nil {
		return r, err
	}

	if dataIndex == -1 || dataIndex >= len(datasets) {
		return r, nil
	}

	tc, err := dstest.NewTestCaseFromDir(TestdataPath(datasets[dataIndex]))
	if err != nil {
		return r, err
	}

	if _, err := createDataset(r, tc); err != nil {
		return nil, fmt.Errorf("error creating dataset: %s", err.Error())
	}
	return r, nil
}

// it's tempting to use base.CreateDataset here, but we can't b/c import cycle :/
// this version of createDataset doesn't run transforms or prepare viz. Test cases
// should be designed to avoid requiring Tranforms be run or Viz be prepped
func createDataset(r repo.Repo, tc dstest.TestCase) (ref reporef.DatasetRef, err error) {
	var (
		ctx = context.Background()
		ds  = tc.Input
		pro *profile.Profile
		// NOTE - struct fields need to be instantiated to make assign set to
		// new pointer values
		userSet = &dataset.Dataset{
			Commit:    &dataset.Commit{},
			Meta:      &dataset.Meta{},
			Structure: &dataset.Structure{},
			Transform: &dataset.Transform{},
			Viz:       &dataset.Viz{},
		}
		path    string
		resBody qfs.File
	)
	pro, err = r.Profile(ctx)
	if err != nil {
		return
	}
	dsName := ds.Name

	userSet.Assign(ds)

	if ds.Commit != nil {
		// NOTE: add author ProfileID here to keep the dataset package agnostic to
		// all identity stuff except keypair crypto
		ds.Commit.Author = &dataset.User{ID: pro.ID.String()}
	}

	sw := dsfs.SaveSwitches{Pin: true, ShouldRender: true}
	fs := r.Filesystem()
	if path, err = dsfs.CreateDataset(ctx, fs, fs.DefaultWriteFS(), r.Bus(), ds, nil, r.PrivateKey(ctx), sw); err != nil {
		return
	}
	if ds.PreviousPath != "" && ds.PreviousPath != "/" {
		prev := reporef.DatasetRef{
			ProfileID: pro.ID,
			Peername:  pro.Peername,
			Name:      dsName,
			Path:      ds.PreviousPath,
		}

		// should be ok to skip this error. we may not have the previous
		// reference locally
		_ = r.DeleteRef(prev)
	}
	ref = reporef.DatasetRef{
		ProfileID: pro.ID,
		Peername:  pro.Peername,
		Name:      dsName,
		Path:      path,
	}

	if err = r.PutRef(ref); err != nil {
		return
	}

	// TODO (b5): confirm these assignments happen in dsfs.CreateDataset with tests
	ds.ProfileID = pro.ID.String()
	ds.Peername = pro.Peername
	ds.Path = path

	// TODO(dustmop): When we switch to initIDs, use the standard resolver instead of logbook.
	// Whether there is a previous version is equivalent to whether we have an initID coming
	// into this function.
	initID, err := r.Logbook().RefToInitID(dsref.Ref{Username: ds.Peername, Name: dsName})
	if err == logbook.ErrNotFound {
		// If dataset does not exist yet, initialize with the given name
		initID, err = r.Logbook().WriteDatasetInit(ctx, dsName)
		if err != nil {
			return ref, err
		}
	}
	if err = r.Logbook().WriteVersionSave(ctx, initID, ds); err != nil && err != logbook.ErrNoLogbook {
		return
	}

	ds, err = dsfs.LoadDataset(ctx, r.Filesystem(), ref.Path)
	if err != nil {
		return ref, err
	}
	ds.Name = ref.Name
	ds.Peername = ref.Peername
	ref.Dataset = ds

	// need to open here b/c we might be doing a dry-run, which would mean we have
	// references to files in a store that won't exist after this function call
	// TODO (b5): this should be replaced with a call to OpenDataset with a qfs that
	// knows about the store
	if resBody, err = r.Filesystem().Get(ctx, ref.Dataset.BodyPath); err != nil {
		return ref, err
	}
	ref.Dataset.SetBodyFile(resBody)

	return ref, nil
}

// NewMemRepoFromDir reads a director of testCases and calls createDataset
// on each case with the given privatekey, yeilding a repo where the peer with
// this pk has created each dataset in question
func NewMemRepoFromDir(path string) (repo.Repo, crypto.PrivKey, error) {
	ctx := context.TODO()
	// TODO (b5) - use a function & a contstant to get this path
	cfgPath := filepath.Join(path, "config.yaml")

	cfg, err := config.ReadFromFile(cfgPath)
	if err != nil {
		return nil, nil, err
	}

	pro, err := profile.NewProfile(cfg.Profile)
	if err != nil {
		return nil, nil, err
	}

	mr, err := repo.NewMemRepoWithProfile(ctx, pro, newTestFS(ctx), event.NilBus)
	if err != nil {
		return mr, pro.PrivKey, err
	}

	tc, err := dstest.LoadTestCases(path)
	if err != nil {
		return mr, pro.PrivKey, err
	}

	for _, c := range tc {
		if _, err := createDataset(mr, c); err != nil {
			return mr, pro.PrivKey, err
		}
	}

	return mr, pro.PrivKey, nil
}

// BadBodyFile is a bunch of bad CSV data
var BadBodyFile = qfs.NewMemfileBytes("bad_csv_file.csv", []byte(`
asdlkfasd,,
fm as
f;lajsmf 
a
's;f a'
sdlfj asdf`))

// BadDataFormatFile has weird line lengths
var BadDataFormatFile = qfs.NewMemfileBytes("abc.csv", []byte(`
"colA","colB","colC","colD"
1,2,3,4
1,2,3`))

// BadStructureFile has double-named columns
var BadStructureFile = qfs.NewMemfileBytes("badStructure.csv", []byte(`
colA, colB, colB, colC
1,2,3,4
1,2,3,4`))
