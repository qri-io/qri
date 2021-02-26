package key

import (
	"encoding/base64"
	"testing"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
)

func TestIDFromPriv(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	expect := "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"
	got, err := IDFromPriv(tr.AlicePrivKey)
	if err != nil {
		t.Error(err)
	}

	if expect != got {
		t.Errorf("ID mismatch. expected: '%s', got: '%s'", expect, got)
	}
}

func TestIDFromPub(t *testing.T) {
	tr, cleanup := newTestRunner(t)
	defer cleanup()

	if _, err := IDFromPub(nil); err == nil {
		t.Errorf("expected error calculating the ID of nil")
	}

	expect := "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"
	got, err := IDFromPub(tr.AlicePrivKey.GetPublic())
	if err != nil {
		t.Error(err)
	}

	if expect != got {
		t.Errorf("ID mismatch. expected: '%s', got: '%s'", expect, got)
	}
}

type testRunner struct {
	AlicePrivKey crypto.PrivKey
	BasitPrivKey crypto.PrivKey
}

func newTestRunner(t *testing.T) (tr *testRunner, cleanup func()) {
	tr = &testRunner{
		AlicePrivKey: testAlicePrivKey(t),
		BasitPrivKey: testBasitPrivKey(t),
	}

	cleanup = func() {

	}
	return tr, cleanup
}

func testAlicePrivKey(t *testing.T) crypto.PrivKey {
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

func testBasitPrivKey(t *testing.T) crypto.PrivKey {
	// id: "QmTqawxrPeTRUKS4GSUURaC16o4etPSJv7Akq6a9xqGZUh"
	testPk := `CAASpwkwggSjAgEAAoIBAQDACiqtbAeIR0gKZZfWuNgDssXnQnEQNrAlISlNMrtULuCtsLBk2tZ4C508T4/JQHfvbazZ/aPvkhr9KBaH8AzDU3FngHQnWblGtfm/0FAXbXPfn6DZ1rbA9rx9XpVZ+pUBDve0YxTSPOo5wOOR9u30JEvO47n1R/bF+wtMRHvDyRuoy4H86XxwMR76LYbgSlJm6SSKnrAVoWR9zqjXdaF1QljO77VbivnR5aS9vQ5Sd1mktwgb3SYUMlEGedtcMdLd3MPVCLFzq6cdjhSwVAxZ3RowR2m0hSEE/p6CKH9xz4wkMmjVrADfQTYU7spym1NBaNCrW1f+r4ScDEqI1yynAgMBAAECggEAWuJ04C5IQk654XHDMnO4h8eLsa7YI3w+UNQo38gqr+SfoJQGZzTKW3XjrC9bNTu1hzK4o1JOy4qyCy11vE/3Olm7SeiZECZ+cOCemhDUVsIOHL9HONFNHHWpLwwcUsEs05tpz400xWrezwZirSnX47tpxTgxQcwVFg2Bg07F5BntepqX+Ns7s2XTEc7YO8o77viYbpfPSjrsToahWP7ngIL4ymDjrZjgWTPZC7AzobDbhjTh5XuVKh60eUz0O7/Ezj2QK00NNkkD7nplU0tojZF10qXKCbECPn3pocVPAetTkwB1Zabq2tC2Y10dYlef0B2fkktJ4PAJyMszx4toQQKBgQD+69aoMf3Wcbw1Z1e9IcOutArrnSi9N0lVD7X2B6HHQGbHkuVyEXR10/8u4HVtbM850ZQjVnSTa4i9XJAy98FWwNS4zFh3OWVhgp/hXIetIlZF72GEi/yVFBhFMcKvXEpO/orEXMOJRdLb/7kNpMvl4MQ/fGWOmQ3InkKxLZFJ+wKBgQDA2jUTvSjjFVtOJBYVuTkfO1DKRGu7QQqNeF978ZEoU0b887kPu2yzx9pK0PzjPffpfUsa9myDSu7rncBX1FP0gNmSIAUja2pwMvJDU2VmE3Ua30Z1gVG1enCdl5ZWufum8Q+0AUqVkBdhPxw+XDJStA95FUArJzeZ2MTwbZH0RQKBgDG188og1Ys36qfPW0C6kNpEqcyAfS1I1rgLtEQiAN5GJMTOVIgF91vy11Rg2QVZrp9ryyOI/HqzAZtLraMCxWURfWn8D1RQkQCO5HaiAKM2ivRgVffvBHZd0M3NglWH/cWhxZW9MTRXtWLJX2DVvh0504s9yuAf4Jw6oG7EoAx5AoGBAJluAURO/jSMTTQB6cAmuJdsbX4+qSc1O9wJpI3LRp06hAPDM7ycdIMjwTw8wLVaG96bXCF7ZCGggCzcOKanupOP34kuCGiBkRDqt2tw8f8gA875S+k4lXU4kFgQvf8JwHi02LVxQZF0LeWkfCfw2eiKcLT4fzDV5ppzp1tREQmxAoGAGOXFomnMU9WPxJp6oaw1ZimtOXcAGHzKwiwQiw7AAWbQ+8RrL2AQHq0hD0eeacOAKsh89OXsdS9iW9GQ1mFR3FA7Kp5srjCMKNMgNSBNIb49iiG9O6P6UcO+RbYGg3CkSTG33W8l2pFIjBrtGktF5GoJudAPR4RXhVsRYZMiGag=`
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
