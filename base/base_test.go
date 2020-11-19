package base

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/muxfs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	repotest "github.com/qri-io/qri/repo/test"
)

// base64-encoded Test Private Key, decoded in init
// peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
var (
	testPk  = []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)
	privKey crypto.PrivKey

	testPeerProfile = &profile.Profile{
		Peername: "peer",
		ID:       profile.IDB58MustDecode("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"),
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

func newTestRepo(t *testing.T) repo.Repo {
	ctx := context.TODO()
	mux, err := muxfs.New(ctx, []qfs.Config{
		{Type: "mem"},
	})
	if err != nil {
		t.Fatal(err)
	}

	mr, err := repo.NewMemRepo(ctx, testPeerProfile, mux, event.NilBus)
	if err != nil {
		t.Fatal(err.Error())
	}
	return mr
}

func addCitiesDataset(t *testing.T, r repo.Repo) dsref.Ref {
	t.Helper()
	prevTS := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }
	defer func() { dsfs.Timestamp = prevTS }()

	ctx := context.Background()
	tc, err := dstest.NewTestCaseFromDir(repotest.TestdataPath("cities"))
	if err != nil {
		t.Fatal(err.Error())
	}

	ds, err := CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), tc.Input, nil, SaveSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err.Error())
	}

	return dsref.ConvertDatasetToVersionInfo(ds).SimpleRef()
}

func updateCitiesDataset(t *testing.T, r repo.Repo, title string) dsref.Ref {
	t.Helper()
	prevTS := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }
	defer func() { dsfs.Timestamp = prevTS }()

	ctx := context.Background()
	tc, err := dstest.NewTestCaseFromDir(repotest.TestdataPath("cities"))
	if err != nil {
		t.Fatal(err.Error())
	}
	pro, err := r.Profile()
	if err != nil {
		t.Fatal(err)
	}

	ref, err := repo.GetVersionInfoShim(r, dsref.Ref{Username: pro.Peername, Name: tc.Name})
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

	res, err := CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), tc.Input, nil, SaveSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err.Error())
	}
	return dsref.ConvertDatasetToVersionInfo(res).SimpleRef()
}

func addFlourinatedCompoundsDataset(t *testing.T, r repo.Repo) dsref.Ref {
	ctx := context.Background()
	tc, err := dstest.NewTestCaseFromDir(repotest.TestdataPath("flourinated_compounds_in_fast_food_packaging"))
	if err != nil {
		t.Fatal(err.Error())
	}

	ref, err := CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), tc.Input, nil, SaveSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err.Error())
	}
	return dsref.ConvertDatasetToVersionInfo(ref).SimpleRef()
}

func addNowTransformDataset(t *testing.T, r repo.Repo) dsref.Ref {
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
		Readme:    &dataset.Readme{},
	}

	script := `
load("time.star", "time")

def transform(ds, ctx):
	ds.set_body([str(time.now())])`
	ds.Transform.SetScriptFile(qfs.NewMemfileBytes("transform.star", []byte(script)))
	ds.SetBodyFile(qfs.NewMemfileBytes("data.json", []byte("[]")))

	readme := "# Oh hey there!\nI'm a readme! hello!\n"
	ds.Readme.SetScriptFile(qfs.NewMemfileBytes("readme.md", []byte(readme)))

	saved, err := CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), ds, nil, SaveSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err.Error())
	}
	return dsref.ConvertDatasetToVersionInfo(saved).SimpleRef()
}

func addTurnstileDataset(t *testing.T, r repo.Repo) dsref.Ref {
	ctx := context.Background()

	ds := &dataset.Dataset{
		Name:     "turnstile_daily_counts_2020",
		Peername: "peer",
		Commit: &dataset.Commit{
			Title: "update data for week ending April 18, 2020",
		},
		Meta: &dataset.Meta{
			Title:       "Turnstile Daily Counts 2020",
			Description: "NYC Subway Turnstile Counts Data aggregated by day and station complex for the year 2020. Updated weekly.",
		},
		Structure: &dataset.Structure{
			Format: "json",
			Schema: dataset.BaseSchemaArray,
		},
		Transform: &dataset.Transform{},
		Readme:    &dataset.Readme{},
	}

	script := `
load("time.star", "time")

def transform(ds, ctx):
	ds.set_body([str(time.now())])`
	ds.Transform.SetScriptFile(qfs.NewMemfileBytes("transform.star", []byte(script)))
	ds.SetBodyFile(qfs.NewMemfileBytes("data.json", []byte("[]")))

	readme := `# nyc-transit-data/turnstile_daily_counts_2020

NYC Subway Turnstile Counts Data aggregated by day and station complex for the year 2020.  Updated weekly.

## Where the Data Came From

This aggregation was created from weekly raw turnstile counts published by the New York MTA at [http://web.mta.info/developers/turnstile.html](http://web.mta.info/developers/turnstile.html)

The raw data were imported into a postgresql database for processing, and aggregated to calendar days for each station complex.

The process is outlined in [this blog post](https://medium.com/qri-io/taming-the-mtas-unruly-turnstile-data-c945f5f96ba0), and the code for the data pipeline is [available on github](https://github.com/qri-io/data-stories-scripts/tree/master/nyc-turnstile-counts).

## Caveats

This aggregation is a best-effort to make a clean and usable dataset of station-level counts.  There were some assumptions and important decisions made to arrive at the finished product.

- The dataset excludes turnstile observation windows (4 hours)  that resulted in entries or exits of over 10,000.  This threshold excludes the obviously spurious numbers that come from the counters rolling over, but could include false readings that are within the threshold.

- The turnstile counts were aggregated to calendar day using the timestamp of the *end* of the 4-hour observation window + 2 hours.  An observation window that ends at 2am would count for the same day, but a window ending between midnight and 1:59am would count for the previous day.

- The last date in the dataset contains a small number of entries and exits that will be aggregated into the next week's worth of data, and should not be used.

## PATH and Roosevelt Island Tramway

The dataset also includes turnstile counts for the PATH train system and the Roosevelt Island Tramway

## Spurious Data in early versions

Versions prior to QmPkGqJ318gcok69Noj3gw3coby8FDrab3x1hBisFcU3Yq were built with a pipeline that had a major error, causing inaccurate numbers near the transition between weekly input files.`
	ds.Readme.SetScriptFile(qfs.NewMemfileBytes("readme.md", []byte(readme)))

	ref, err := CreateDataset(ctx, r, r.Filesystem().DefaultWriteFS(), ds, nil, SaveSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err.Error())
	}

	return dsref.ConvertDatasetToVersionInfo(ref).SimpleRef()
}
