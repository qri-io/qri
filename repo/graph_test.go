package repo

import (
	"encoding/base64"
	"fmt"
	"github.com/qri-io/dataset/dsfs"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/repo/profile"
)

// base64-encoded Test Private Key, decoded in init
// peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
var testPk = []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)

func init() {
	data, err := base64.StdEncoding.DecodeString(string(testPk))
	if err != nil {
		panic(err)
	}
	testPk = data
}

func TestGraph(t *testing.T) {
	r, err := makeTestRepo()
	if err != nil {
		t.Errorf("error making test repo: %s", err.Error())
		return
	}
	nodes, err := Graph(r)
	if err != nil {
		t.Errorf("error generating repo graph: %s", err.Error())
		return
	}

	expect := 8
	count := 0
	for range nodes {
		count++
	}
	if count != expect {
		t.Errorf("node count mismatch. expected: %d, got: %d", expect, count)
	}
}

func TestQueriesMap(t *testing.T) {
	r, err := makeTestRepo()
	if err != nil {
		t.Errorf("error making test repo: %s", err.Error())
		return
	}
	node, err := Graph(r)
	if err != nil {
		t.Errorf("error generating repo graph: %s", err.Error())
		return
	}

	qs := QueriesMap(node)
	expect := 1
	if len(qs) != expect {
		t.Errorf("query count mismatch, expected: %d, got: %d", expect, len(qs))
	}
}

func TestDatasetQueries(t *testing.T) {
	r, err := makeTestRepo()
	if err != nil {
		t.Errorf("error making test repo: %s", err.Error())
		return
	}
	node, err := Graph(r)
	if err != nil {
		t.Errorf("error generating repo graph: %s", err.Error())
		return
	}

	qs := DatasetQueries(node)
	expect := 2
	if len(qs) != expect {
		t.Errorf("query count mismatch, expected: %d, got: %d", expect, len(qs))
	}
}

func TestDataNodes(t *testing.T) {
	r, err := makeTestRepo()
	if err != nil {
		t.Errorf("error making test repo: %s", err.Error())
		return
	}
	node, err := Graph(r)
	if err != nil {
		t.Errorf("error generating repo graph: %s", err.Error())
		return
	}

	dn := DataNodes(node)
	expect := 2
	if len(dn) != expect {
		t.Errorf("data node mismatch, expected: %d, got: %d", expect, len(dn))
	}
}

func makeTestRepo() (Repo, error) {
	ds1 := &dataset.Dataset{
		Meta: &dataset.Meta{
			Title: "dataset 1",
		},
		Commit: &dataset.Commit{
			Message: "foo",
		},
		PreviousPath: "",
		Structure: &dataset.Structure{
			Format: dataset.JSONDataFormat,
			Schema: dataset.BaseSchemaObject,
		},
	}
	ds2 := &dataset.Dataset{
		Meta: &dataset.Meta{
			Title: "dataset 2",
		},
		Commit: &dataset.Commit{
			Message: "bar",
		},
		Structure: &dataset.Structure{
			Format: dataset.JSONDataFormat,
			Schema: dataset.BaseSchemaObject,
		},
		Transform: &dataset.Transform{
			Syntax: "sql",
			Data:   "select * from a,b where b.id = 'foo'",
			Resources: map[string]*dataset.Dataset{
				"a": dataset.NewDatasetRef(datastore.NewKey("/path/to/a")),
				"b": dataset.NewDatasetRef(datastore.NewKey("/path/to/b")),
			},
		},
		PreviousPath: "",
	}
	store := cafs.NewMapstore()
	p := &profile.Profile{}

	r, err := NewMemRepo(p, store, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating test repo: %s", err.Error())
	}

	privKey, err := crypto.UnmarshalPrivateKey(testPk)
	if err != nil {
		err = fmt.Errorf("error unmarshaling private key: %s", err.Error())
		return nil, err
	}

	r.SetPrivateKey(privKey)
	r.SetProfile(&profile.Profile{
		ID:       "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		Peername: "peer",
	})

	data1f := cafs.NewMemfileBytes("data1", []byte("dataset_1"))

	ds1p, err := dsfs.WriteDataset(store, ds1, data1f, true)
	if err != nil {
		return nil, fmt.Errorf("error putting dataset: %s", err.Error())
	}
	r.PutRef(DatasetRef{Peername: "peer", Name: "ds1", Path: ds1p.String()})

	data2f := cafs.NewMemfileBytes("data2", []byte("dataset_2"))
	ds2p, err := dsfs.WriteDataset(store, ds2, data2f, true)
	if err != nil {
		return nil, fmt.Errorf("error putting dataset: %s", err.Error())
	}
	r.PutRef(DatasetRef{Peername: "peer", Name: "ds2", Path: ds2p.String()})

	return r, nil
}
