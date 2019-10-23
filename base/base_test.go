package base

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

// base64-encoded Test Private Key, decoded in init
// peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
var (
	testPk  = []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)
	privKey crypto.PrivKey

	testPeerProfile = &profile.Profile{
		Peername: "peer",
		ID:       "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
	}

	devNull = ioes.NewDiscardIOStreams()
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

func testdataPath(path string) string {
	return filepath.Join(os.Getenv("GOPATH"), "/src/github.com/qri-io/qri/repo/test/testdata", path)
}

func newTestRepo(t *testing.T) repo.Repo {
	mapStore := cafs.NewMapstore()
	mux := qfs.NewMux(map[string]qfs.Filesystem{
		"map":  mapStore,
		"cafs": mapStore,
	})
	mr, err := repo.NewMemRepo(testPeerProfile, mapStore, mux, profile.NewMemStore())
	if err != nil {
		t.Fatal(err.Error())
	}
	return mr
}

func addCitiesDataset(t *testing.T, r repo.Repo) repo.DatasetRef {
	ctx := context.Background()
	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
	if err != nil {
		t.Fatal(err.Error())
	}

	ref, err := CreateDataset(ctx, r, ioes.NewDiscardIOStreams(), tc.Input, nil, false, true, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	return ref
}

func updateCitiesDataset(t *testing.T, r repo.Repo, title string) repo.DatasetRef {
	ctx := context.Background()
	tc, err := dstest.NewTestCaseFromDir(testdataPath("cities"))
	if err != nil {
		t.Fatal(err.Error())
	}
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

	ref, err := r.GetRef(repo.DatasetRef{Peername: pro.Peername, Name: tc.Name})
	if err != nil {
		t.Fatal(err)
	}

	if title == "" {
		title = "this is the new title"
	}

	prevTitle := tc.Input.Meta.Title
	tc.Input.Meta.Title = title
	tc.Input.PreviousPath = ref.Path
	defer func() {
		// because test cases are cached for performance, we need to clean up any mutation to
		// testcase input
		tc.Input.Meta.Title = prevTitle
		tc.Input.PreviousPath = ""
	}()

	ref, err = CreateDataset(ctx, r, devNull, tc.Input, nil, false, true, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	return ref
}

func addFlourinatedCompoundsDataset(t *testing.T, r repo.Repo) repo.DatasetRef {
	ctx := context.Background()
	tc, err := dstest.NewTestCaseFromDir(testdataPath("flourinated_compounds_in_fast_food_packaging"))
	if err != nil {
		t.Fatal(err.Error())
	}

	ref, err := CreateDataset(ctx, r, devNull, tc.Input, nil, false, true, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	return ref
}

func addNowTransformDataset(t *testing.T, r repo.Repo) repo.DatasetRef {
	ctx := context.Background()

	ds := &dataset.Dataset{
		Name:     "now_tf",
		Peername: "peer",
		Commit: &dataset.Commit{
			Title: "",
		},
		Meta: &dataset.Meta{
			Title: "example transform",
		},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
		Transform: &dataset.Transform{},
	}

	script := `
load("time.star", "time")

def transform(ds, ctx):
	ds.set_body([str(time.now())])`
	ds.Transform.SetScriptFile(qfs.NewMemfileBytes("transform.star", []byte(script)))
	ds.SetBodyFile(qfs.NewMemfileBytes("data.json", []byte("[]")))

	ref, err := CreateDataset(ctx, r, devNull, ds, nil, false, true, false, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	return ref
}
