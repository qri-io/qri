package regclient

import (
	"encoding/base64"
	"fmt"
	"net/http/httptest"
	"testing"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver/handlers"
)

// base64-encoded Test Private Key, decoded in init
// peerId: QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt
var (
	pk1Bytes = []byte(`CAASpgkwggSiAgEAAoIBAQC/7Q7fILQ8hc9g07a4HAiDKE4FahzL2eO8OlB1K99Ad4L1zc2dCg+gDVuGwdbOC29IngMA7O3UXijycckOSChgFyW3PafXoBF8Zg9MRBDIBo0lXRhW4TrVytm4Etzp4pQMyTeRYyWR8e2hGXeHArXM1R/A/SjzZUbjJYHhgvEE4OZy7WpcYcW6K3qqBGOU5GDMPuCcJWac2NgXzw6JeNsZuTimfVCJHupqG/dLPMnBOypR22dO7yJIaQ3d0PFLxiDG84X9YupF914RzJlopfdcuipI+6gFAgBw3vi6gbECEzcohjKf/4nqBOEvCDD6SXfl5F/MxoHurbGBYB2CJp+FAgMBAAECggEAaVOxe6Y5A5XzrxHBDtzjlwcBels3nm/fWScvjH4dMQXlavwcwPgKhy2NczDhr4X69oEw6Msd4hQiqJrlWd8juUg6vIsrl1wS/JAOCS65fuyJfV3Pw64rWbTPMwO3FOvxj+rFghZFQgjg/i45uHA2UUkM+h504M5Nzs6Arr/rgV7uPGR5e5OBw3lfiS9ZaA7QZiOq7sMy1L0qD49YO1ojqWu3b7UaMaBQx1Dty7b5IVOSYG+Y3U/dLjhTj4Hg1VtCHWRm3nMOE9cVpMJRhRzKhkq6gnZmni8obz2BBDF02X34oQLcHC/Wn8F3E8RiBjZDI66g+iZeCCUXvYz0vxWAQQKBgQDEJu6flyHPvyBPAC4EOxZAw0zh6SF/r8VgjbKO3n/8d+kZJeVmYnbsLodIEEyXQnr35o2CLqhCvR2kstsRSfRz79nMIt6aPWuwYkXNHQGE8rnCxxyJmxV4S63GczLk7SIn4KmqPlCI08AU0TXJS3zwh7O6e6kBljjPt1mnMgvr3QKBgQD6fAkdI0FRZSXwzygx4uSg47Co6X6ESZ9FDf6ph63lvSK5/eue/ugX6p/olMYq5CHXbLpgM4EJYdRfrH6pwqtBwUJhlh1xI6C48nonnw+oh8YPlFCDLxNG4tq6JVo071qH6CFXCIank3ThZeW5a3ZSe5pBZ8h4bUZ9H8pJL4C7yQKBgFb8SN/+/qCJSoOeOcnohhLMSSD56MAeK7KIxAF1jF5isr1TP+rqiYBtldKQX9bIRY3/8QslM7r88NNj+aAuIrjzSausXvkZedMrkXbHgS/7EAPflrkzTA8fyH10AsLgoj/68mKr5bz34nuY13hgAJUOKNbvFeC9RI5g6eIqYH0FAoGAVqFTXZp12rrK1nAvDKHWRLa6wJCQyxvTU8S1UNi2EgDJ492oAgNTLgJdb8kUiH0CH0lhZCgr9py5IKW94OSM6l72oF2UrS6PRafHC7D9b2IV5Al9lwFO/3MyBrMocapeeyaTcVBnkclz4Qim3OwHrhtFjF1ifhP9DwVRpuIg+dECgYANwlHxLe//tr6BM31PUUrOxP5Y/cj+ydxqM/z6papZFkK6Mvi/vMQQNQkh95GH9zqyC5Z/yLxur4ry1eNYty/9FnuZRAkEmlUSZ/DobhU0Pmj8Hep6JsTuMutref6vCk2n02jc9qYmJuD7iXkdXDSawbEG6f5C4MUkJ38z1t1OjA==`)
	pk1      crypto.PrivKey
)

func init() {
	data, err := base64.StdEncoding.DecodeString(string(pk1Bytes))
	if err != nil {
		panic(err)
	}
	pk1Bytes = data

	pk1, err = crypto.UnmarshalPrivateKey(pk1Bytes)
	if err != nil {
		panic(fmt.Errorf("error unmarshaling private key: %s", err.Error()))
	}
}

func TestProfileRequests(t *testing.T) {
	input := &registry.Profile{
		Username: "b5",
	}

	reg := registry.Registry{
		Profiles: registry.NewMemProfiles(),
		Search:   &registry.MockSearch{},
	}
	ts := httptest.NewServer(handlers.NewRoutes(reg))
	c := NewClient(&Config{
		Location: ts.URL,
	})

	pubBytes, err := pk1.GetPublic().Bytes()
	if err != nil {
		t.Error(err.Error())
	}

	p := &registry.Profile{
		PublicKey: base64.StdEncoding.EncodeToString(pubBytes),
	}

	err = c.GetProfile(p)
	if err == nil {
		t.Errorf("expected empty get to error")
	} else if err.Error() != "error 404: " {
		t.Errorf("error mistmatch. expected: %s, got: %s", "error 404: ", err.Error())
	}

	_, err = c.PutProfile(input, pk1)
	if err != nil {
		t.Error(err.Error())
	}

	err = c.GetProfile(p)
	if err != nil {
		t.Error(err)
	}
	if p.Username != input.Username {
		t.Errorf("expected username to equal %s, got: %s", input.Username, p.Username)
	}

	err = c.DeleteProfile(input, pk1)
	if err != nil {
		t.Error(err.Error())
	}
}
