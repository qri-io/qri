package log

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/google/go-cmp/cmp"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/qri/logbook/log/logfb"
)

var allowUnexported = cmp.AllowUnexported(
	Book{},
	Log{},
	Op{},
)

func TestBookFlatbuffer(t *testing.T) {
	pk := testPrivKey(t)
	log := InitLog(Op{
		Type:      OpTypeInit,
		Model:     0x0001,
		Ref:       "QmRefHash",
		Prev:      "QmPrevHash",
		Relations: []string{"a", "b", "c"},
		Name:      "steve",
		AuthorID:  "QmSteveHash",
		Timestamp: 1,
		Size:      2,
		Note:      "note!",
	})
	log.signature = []byte{1, 2, 3}

	log.AddChild(InitLog(Op{
		Type:      OpTypeInit,
		Model:     0x0002,
		Ref:       "QmRefHash",
		Name:      "steve",
		AuthorID:  "QmSteveHash",
		Timestamp: 2,
		Size:      2500000,
		Note:      "note?",
	}))

	book := &Book{
		pk:         pk,
		authorname: "must_preserve",
		logs: map[uint32][]*Log{
			0x0001: []*Log{log},
		},
	}

	data := book.flatbufferBytes()
	logsetfb := logfb.GetRootAsBook(data, 0)

	got := &Book{
		// re-provide private key, unmarshal flatbuffer must preserve this key
		// through the unmarshaling call
		pk: pk,
	}
	if err := got.unmarshalFlatbuffer(logsetfb); err != nil {
		t.Fatalf("unmarshalling flatbuffer bytes: %s", err.Error())
	}

	if diff := cmp.Diff(book, got, allowUnexported, cmp.Comparer(comparePrivKeys)); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestBookCiphertext(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	lg := tr.RandomLog(Op{
		Type:  OpTypeInit,
		Model: 0x0001,
		Name:  "apples",
	}, 10)

	book := tr.Book
	book.AppendLog(lg)

	gotcipher, err := book.FlatbufferCipher()
	if err != nil {
		t.Fatalf("calculating flatbuffer cipher: %s", err.Error())
	}

	plaintext := book.flatbufferBytes()
	if bytes.Equal(gotcipher, plaintext) {
		t.Errorf("plaintext bytes & ciphertext bytes can't be equal")
	}

	// TODO (b5) - we should confirm the ciphertext isn't readable, but
	// this'll panic with out-of-bounds slice access...
	// ciphertextAsBook := logfb.GetRootAsBook(gotcipher, 0)
	// if err := book.unmarshalFlatbuffer(ciphertextAsBook); err == nil {
	// 	t.Errorf("ciphertext as book should not have worked")
	// }

	if err = book.UnmarshalFlatbufferCipher(tr.Ctx, gotcipher); err != nil {
		t.Errorf("book.UnmarhsalFlatbufferCipher unexpected error: %s", err.Error())
	}
}

func TestBookSignLog(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	lg := tr.RandomLog(Op{
		Type:  OpTypeInit,
		Model: 0x0001,
		Name:  "apples",
	}, 400)

	pk := tr.Book.pk
	data, err := lg.SignedFlatbufferBytes(pk)
	if err != nil {
		t.Fatal(err)
	}

	received, err := FromFlatbufferBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	if err := received.Verify(pk.GetPublic()); err != nil {
		t.Fatal(err)
	}
}

type testRunner struct {
	Ctx        context.Context
	AuthorName string
	AuthorID   string
	Book       *Book
	gen        *opGenerator
}

func newTestRunner(t *testing.T) (tr testRunner, cleanup func()) {
	ctx := context.Background()
	authorName := "test_author"
	authorID := "QmTestAuthorID"
	pk := testPrivKey(t)

	book, err := NewBook(pk, authorName, authorID)
	if err != nil {
		t.Fatalf("creating book: %s", err.Error())
	}

	tr = testRunner{
		Ctx:        ctx,
		AuthorName: authorName,
		AuthorID:   authorID,
		Book:       book,
		gen:        &opGenerator{ctx: ctx, NoopProb: 60},
	}
	cleanup = func() {
		// noop
	}

	return tr, cleanup
}

func (tr testRunner) RandomLog(init Op, opCount int) *Log {
	lg := InitLog(init)
	for i := 0; i < opCount; i++ {
		lg.Append(tr.gen.Gen())
	}
	return lg
}

func testPrivKey(t *testing.T) crypto.PrivKey {
	// logbooks are encrypted at rest, we need a private key to interact with
	// them, including to create a new logbook. This is a dummy Private Key
	// you should never, ever use in real life. demo only folks.
	testPk := `CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`
	data, err := base64.StdEncoding.DecodeString(testPk)
	if err != nil {
		t.Fatal(err)
	}
	pk, err := crypto.UnmarshalPrivateKey(data)
	if err != nil {
		t.Fatalf("error unmarshaling private key: %s", err.Error())
	}
	return pk
}

func comparePrivKeys(a, b crypto.PrivKey) bool {
	if a == nil && b != nil || a != nil && b == nil {
		return false
	}

	abytes, err := a.Bytes()
	if err != nil {
		return false
	}

	bbytes, err := b.Bytes()
	if err != nil {
		return false
	}

	return string(abytes) == string(bbytes)
}
