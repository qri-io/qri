package cbornode

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	cid "gx/ipfs/QmV5gPoRsjN1Gid3LMdNZTyfCtP2DsvqEbMAmz82RmmiGk/go-cid"
	u "gx/ipfs/QmZuY8aV7zbNXVy6DyN9SmnuH3o9nG852F4aTiSBpts8d1/go-ipfs-util"
)

func TestBasicMarshal(t *testing.T) {
	c := cid.NewCidV0(u.Hash([]byte("something")))

	obj := map[string]interface{}{
		"name": "foo",
		"bar":  c,
	}

	nd, err := WrapObject(obj)
	if err != nil {
		t.Fatal(err)
	}

	back, err := Decode(nd.RawData())
	if err != nil {
		t.Fatal(err)
	}

	lnk, _, err := back.ResolveLink([]string{"bar"})
	if err != nil {
		t.Fatal(err)
	}

	if !lnk.Cid.Equals(c) {
		t.Fatal("expected cid to match")
	}

	if !nd.Cid().Equals(back.Cid()) {
		t.Fatal("re-serialize failed to generate same cid")
	}
}

func TestMarshalRoundtrip(t *testing.T) {
	c1 := cid.NewCidV0(u.Hash([]byte("something1")))
	c2 := cid.NewCidV0(u.Hash([]byte("something2")))
	c3 := cid.NewCidV0(u.Hash([]byte("something3")))

	obj := map[interface{}]interface{}{
		"foo": "bar",
		"baz": []interface{}{
			c1,
			c2,
		},
		"cats": map[interface{}]interface{}{
			"qux": c3,
		},
	}

	nd1, err := WrapObject(obj)
	if err != nil {
		t.Fatal(err)
	}

	if len(nd1.Links()) != 3 {
		t.Fatal("didnt have enough links")
	}

	nd2, err := Decode(nd1.RawData())
	if err != nil {
		t.Fatal(err)
	}

	if !nd1.Cid().Equals(nd2.Cid()) {
		t.Fatal("objects didnt match between marshalings")
	}

	lnk, rest, err := nd2.ResolveLink([]string{"baz", "1", "bop"})
	if err != nil {
		t.Fatal(err)
	}

	if !lnk.Cid.Equals(c2) {
		t.Fatal("expected c2")
	}

	if len(rest) != 1 || rest[0] != "bop" {
		t.Fatal("should have had one path element remaning after resolve")
	}

	out, err := nd1.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(out))
}

func assertStringsEqual(t *testing.T, a, b []string) {
	if len(a) != len(b) {
		t.Fatal("lengths differed: ", a, b)
	}

	sort.Strings(a)
	sort.Strings(b)

	for i, v := range a {
		if v != b[i] {
			t.Fatal("got mismatch: ", a, b)
		}
	}
}

func TestTree(t *testing.T) {
	c1 := cid.NewCidV0(u.Hash([]byte("something1")))
	c2 := cid.NewCidV0(u.Hash([]byte("something2")))
	c3 := cid.NewCidV0(u.Hash([]byte("something3")))
	c4 := cid.NewCidV0(u.Hash([]byte("something4")))

	obj := map[interface{}]interface{}{
		"foo": c1,
		"baz": []interface{}{c2, c3, "c"},
		"cats": map[interface{}]interface{}{
			"qux": map[interface{}]interface{}{
				"boo": 1,
				"baa": c4,
				"bee": 3,
				"bii": 4,
				"buu": map[interface{}]interface{}{
					"coat": "rain",
				},
			},
		},
	}

	nd, err := WrapObject(obj)
	if err != nil {
		t.Fatal(err)
	}

	full := []string{
		"foo",
		"baz",
		"baz/0",
		"baz/1",
		"baz/2",
		"cats",
		"cats/qux",
		"cats/qux/boo",
		"cats/qux/baa",
		"cats/qux/bee",
		"cats/qux/bii",
		"cats/qux/buu",
		"cats/qux/buu/coat",
	}

	assertStringsEqual(t, full, nd.Tree("", -1))

	cats := []string{
		"qux",
		"qux/boo",
		"qux/baa",
		"qux/bee",
		"qux/bii",
		"qux/buu",
		"qux/buu/coat",
	}

	assertStringsEqual(t, cats, nd.Tree("cats", -1))

	toplevel := []string{
		"foo",
		"baz",
		"cats",
	}

	assertStringsEqual(t, toplevel, nd.Tree("", 1))
}

func TestParsing(t *testing.T) {
	b := []byte("\xd9\x01\x02\x58\x25\xa5\x03\x22\x12\x20\x65\x96\x50\xfc\x34\x43\xc9\x16\x42\x80\x48\xef\xc5\xba\x45\x58\xdc\x86\x35\x94\x98\x0a\x59\xf5\xcb\x3c\x4d\x84\x86\x7e\x6d\x31")

	n, err := Decode(b)
	t.Log(n, err)
}

func TestFromJson(t *testing.T) {
	data := `{
        "something": {"/":"zb2rhisguzLFRJaxg6W3SiToBYgESFRGk1wiCRGJYF9jqk1Uw"},
        "cats": "not cats",
        "cheese": [
                {"/":"zb2rhisguzLFRJaxg6W3SiToBYgESFRGk1wiCRGJYF9jqk1Uw"},
                {"/":"zb2rhisguzLFRJaxg6W3SiToBYgESFRGk1wiCRGJYF9jqk1Uw"},
                {"/":"zb2rhisguzLFRJaxg6W3SiToBYgESFRGk1wiCRGJYF9jqk1Uw"},
                {"/":"zb2rhisguzLFRJaxg6W3SiToBYgESFRGk1wiCRGJYF9jqk1Uw"}
        ]
}`
	n, err := FromJson(bytes.NewReader([]byte(data)))
	if err != nil {
		t.Fatal(err)
	}
	c, ok := n.obj.(map[interface{}]interface{})["something"].(*cid.Cid)
	if !ok {
		t.Fatal("expected a cid")
	}

	if c.String() != "zb2rhisguzLFRJaxg6W3SiToBYgESFRGk1wiCRGJYF9jqk1Uw" {
		t.Fatal("cid unmarshaled wrong")
	}
}

func TestResolvedValIsJsonable(t *testing.T) {
	data := `{
		"foo": {
			"bar": 1,
			"baz": 2
		}
	}`
	n, err := FromJson(strings.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	val, _, err := n.Resolve([]string{"foo"})
	if err != nil {
		t.Fatal(err)
	}

	out, err := json.Marshal(val)
	if err != nil {
		t.Fatal(err)
	}

	if string(out) != `{"bar":1,"baz":2}` {
		t.Fatal("failed to get expected json")
	}
}

func TestExamples(t *testing.T) {
	examples := []string{
		"[null]",
		"[]",
		"{}",
		"null",
		"1",
		"[1]",
		"true",
		`{"a":"IPFS"}`,
		`{"a":"IPFS","b":null,"c":[1]}`,
		`{"a":[]}`,
	}
	for _, originalJson := range examples {
		n, err := FromJson(bytes.NewReader([]byte(originalJson)))
		if err != nil {
			t.Fatal(err)
		}

		cbor := n.RawData()
		node, err := Decode(cbor)
		if err != nil {
			t.Fatal(err)
		}

		node, err = Decode(cbor)
		if err != nil {
			t.Fatal(err)
		}

		jsonBytes, err := node.MarshalJSON()
		json := string(jsonBytes)
		if json != originalJson {
			t.Fatal("marshaled to incorrect JSON: " + json)
		}
	}
}
