package test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qfs/httpfs"
	"github.com/qri-io/qfs/localfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
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
func NewEmptyTestRepo() (mr *repo.MemRepo, err error) {
	pro := &profile.Profile{
		Peername: "peer",
		ID:       profile.IDB58MustDecode(profileID),
		PrivKey:  privKey,
	}
	ms := cafs.NewMapstore()
	return repo.NewMemRepo(pro, ms, newTestFS(ms), profile.NewMemStore())
}

func newTestFS(cafsys cafs.Filestore) qfs.Filesystem {
	return qfs.NewMux(map[string]qfs.Filesystem{
		"local": localfs.NewFS(),
		"http":  httpfs.NewFS(),
		"cafs":  cafsys,
	})
}

// NewTestRepo generates a repository usable for testing purposes
func NewTestRepo() (mr *repo.MemRepo, err error) {
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	mr, err = NewEmptyTestRepo()
	if err != nil {
		return
	}

	gopath := os.Getenv("GOPATH")
	for _, k := range datasets {
		tc, err := dstest.NewTestCaseFromDir(fmt.Sprintf("%s/src/github.com/qri-io/qri/repo/test/testdata/%s", gopath, k))
		if err != nil {
			return nil, err
		}
		if _, err := createDataset(mr, tc); err != nil {
			return nil, fmt.Errorf("%s error creating dataset: %s", k, err.Error())
		}
	}

	return
}

// NewTestRepoWithHistory generates a repository with a dataset that has a history, usable for testing purposes
func NewTestRepoWithHistory() (mr *repo.MemRepo, refs []repo.DatasetRef, err error) {
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	mr, err = NewEmptyTestRepo()
	if err != nil {
		return
	}

	gopath := os.Getenv("GOPATH")
	prevPath := ""
	for _, k := range datasets {
		tc, err := dstest.NewTestCaseFromDir(fmt.Sprintf("%s/src/github.com/qri-io/qri/repo/test/testdata/%s", gopath, k))
		if err != nil {
			return nil, nil, err
		}
		tc.Input.Name = "logtest"
		tc.Input.PreviousPath = prevPath
		ref, err := createDataset(mr, tc)
		if err != nil {
			return nil, nil, fmt.Errorf("%s error creating dataset: %s", k, err.Error())
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
	datasets := []string{"movies", "cities", "counter", "craigslist", "sitemap"}

	pk, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	ms := cafs.NewMapstore()
	r, err := repo.NewMemRepo(&profile.Profile{
		ID:       id,
		Peername: fmt.Sprintf("test-repo-%d", peerNum),
		PrivKey:  pk,
	}, ms, newTestFS(ms), profile.NewMemStore())
	if err != nil {
		return r, err
	}

	if dataIndex == -1 || dataIndex >= len(datasets) {
		return r, nil
	}

	gopath := os.Getenv("GOPATH")
	filepath := fmt.Sprintf("%s/src/github.com/qri-io/qri/repo/test/testdata/%s", gopath, datasets[dataIndex])
	tc, err := dstest.NewTestCaseFromDir(filepath)
	if err != nil {
		return r, err
	}

	if _, err := createDataset(r, tc); err != nil {
		return nil, fmt.Errorf("error creating dataset: %s", err.Error())
	}
	return r, nil
}

func pkgPath(paths ...string) string {
	gp := os.Getenv("GOPATH")
	return filepath.Join(append([]string{gp, "src/github.com/qri-io/qri/repo/test"}, paths...)...)
}

// it's tempting to use actions.Dataset.CreateDataset here, but we can't b/c import cycle :/
// this version of createDataset doesn't run transforms or prepare viz. Test cases
// should be designed to avoid requiring Tranforms be run or Viz be prepped
func createDataset(r repo.Repo, tc dstest.TestCase) (ref repo.DatasetRef, err error) {
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
	)
	pro, err = r.Profile()
	if err != nil {
		return
	}

	userSet.Assign(ds)

	if ds.Commit != nil {
		// NOTE: add author ProfileID here to keep the dataset package agnostic to
		// all identity stuff except keypair crypto
		ds.Commit.Author = &dataset.User{ID: pro.ID.String()}
	}

	ref, err = base.CreateDataset(ctx, r, ioes.NewDiscardIOStreams(), ds, nil, false, true, false, true)
	return
}

// NewMemRepoFromDir reads a director of testCases and calls createDataset
// on each case with the given privatekey, yeilding a repo where the peer with
// this pk has created each dataset in question
func NewMemRepoFromDir(path string) (repo.Repo, crypto.PrivKey, error) {
	pro, pk, err := ReadRepoConfig(filepath.Join(path, "config.yaml"))
	if err != nil {
		return nil, pk, err
	}

	ms := cafs.NewMapstore()
	mr, err := repo.NewMemRepo(pro, ms, newTestFS(ms), profile.NewMemStore())
	if err != nil {
		return mr, pk, err
	}

	tc, err := dstest.LoadTestCases(path)
	if err != nil {
		return mr, pk, err
	}

	for _, c := range tc {
		if _, err := createDataset(mr, c); err != nil {
			return mr, pk, err
		}
	}

	return mr, pk, nil
}

// ReadRepoConfig loads configuration data from a .yaml file
func ReadRepoConfig(path string) (pro *profile.Profile, pk crypto.PrivKey, err error) {
	cfgData, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("error reading config file: %s", err.Error())
		return
	}

	cfg := &struct {
		PrivateKey string
		Profile    *profile.Profile
	}{}
	if err = yaml.Unmarshal(cfgData, &cfg); err != nil {
		err = fmt.Errorf("error unmarshaling yaml data: %s", err.Error())
		return
	}

	pro = cfg.Profile

	pkdata, err := base64.StdEncoding.DecodeString(cfg.PrivateKey)
	if err != nil {
		err = fmt.Errorf("invalde privatekey: %s", err.Error())
		return
	}

	pk, err = crypto.UnmarshalPrivateKey(pkdata)
	if err != nil {
		err = fmt.Errorf("error unmarshaling privatekey: %s", err.Error())
		return
	}
	pro.PrivKey = pk

	return
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
