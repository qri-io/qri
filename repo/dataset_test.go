package repo

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset"
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

func TestCreateDataset(t *testing.T) {
	r, err := NewMemRepo(testPeerProfile, cafs.NewMapstore(), profile.NewMemStore(), nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	ds := &dataset.Dataset{
		Meta:   &dataset.Meta{Title: "test"},
		Commit: &dataset.Commit{Title: "hello"},
		Structure: &dataset.Structure{
			Format: dataset.JSONDataFormat,
			Schema: dataset.BaseSchemaArray,
		},
	}

	if _, err := CreateDataset(r, "foo", &dataset.Dataset{}, nil, true); err == nil {
		t.Error("expected bad dataset to error")
	}

	ref, err := CreateDataset(r, "foo", ds, cafs.NewMemfileBytes("body.json", []byte("[]")), true)
	if err != nil {
		t.Fatal(err.Error())
	}
	refs, err := r.References(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Errorf("ref length mismatch. expected 1, got: %d", len(refs))
	}

	ds.Meta.Title = "an update"
	ds.PreviousPath = ref.Path

	ref, err = CreateDataset(r, "foo", ds, cafs.NewMemfileBytes("body.json", []byte("[]")), true)
	if err != nil {
		t.Fatal(err.Error())
	}
	refs, err = r.References(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 {
		t.Errorf("ref length mismatch. expected 1, got: %d", len(refs))
	}
}

func TestDatasetPodBodyFile(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"json":"data"}`))
	}))
	badS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	cases := []struct {
		dsp      *dataset.DatasetPod
		filename string
		fileLen  int
		err      string
	}{
		// bad input
		{&dataset.DatasetPod{}, "", 0, "not found"},

		// inline data
		{&dataset.DatasetPod{BodyBytes: []byte("a,b,c\n1,2,3")}, "", 0, "specifying bodyBytes requires format be specified in dataset.structure"},
		{&dataset.DatasetPod{Structure: &dataset.StructurePod{Format: "csv"}, BodyBytes: []byte("a,b,c\n1,2,3")}, "body.csv", 11, ""},

		// urlz
		{&dataset.DatasetPod{BodyPath: "http://"}, "", 0, "fetching body url: Get http:: http: no Host in request URL"},
		{&dataset.DatasetPod{BodyPath: fmt.Sprintf("%s/foobar.json", badS.URL)}, "", 0, "invalid status code fetching body url: 500"},
		{&dataset.DatasetPod{BodyPath: fmt.Sprintf("%s/foobar.json", s.URL)}, "foobar.json", 15, ""},

		// local filepaths
		{&dataset.DatasetPod{BodyPath: "nope.cbor"}, "", 0, "reading body file: open nope.cbor: no such file or directory"},
		{&dataset.DatasetPod{BodyPath: "nope.yaml"}, "", 0, "reading body file: open nope.yaml: no such file or directory"},
		{&dataset.DatasetPod{BodyPath: "testdata/schools.cbor"}, "schools.cbor", 154, ""},
		{&dataset.DatasetPod{BodyPath: "testdata/bad.yaml"}, "", 0, "converting yaml body to json: yaml: line 1: did not find expected '-' indicator"},
		{&dataset.DatasetPod{BodyPath: "testdata/oh_hai.yaml"}, "oh_hai.json", 29, ""},
	}

	for i, c := range cases {
		file, err := DatasetPodBodyFile(c.dsp)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if file == nil && c.filename != "" {
			t.Errorf("case %d expected file", i)
			continue
		} else if c.filename == "" {
			continue
		}

		if c.filename != file.FileName() {
			t.Errorf("case %d filename mismatch. expected: '%s', got: '%s'", i, c.filename, file.FileName())
		}
		data, err := ioutil.ReadAll(file)
		if err != nil {
			t.Errorf("case %d error reading file: %s", i, err.Error())
			continue
		}
		if c.fileLen != len(data) {
			t.Errorf("case %d file length mismatch. expected: %d, got: %d", i, c.fileLen, len(data))
		}

		if err := file.Close(); err != nil {
			t.Errorf("case %d error closing file: %s", i, err.Error())
		}
	}
}
