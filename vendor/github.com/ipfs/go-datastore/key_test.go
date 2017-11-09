package datastore_test

import (
	"bytes"
	"math/rand"
	"path"
	"strings"
	"testing"

	. "github.com/ipfs/go-datastore"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func randomString() string {
	chars := "abcdefghijklmnopqrstuvwxyz1234567890"
	var buf bytes.Buffer
	l := rand.Intn(50)
	for j := 0; j < l; j++ {
		buf.WriteByte(chars[rand.Intn(len(chars))])
	}
	return buf.String()
}

type KeySuite struct{}

var _ = Suite(&KeySuite{})

func (ks *KeySuite) SubtestKey(s string, c *C) {
	fixed := path.Clean("/" + s)
	namespaces := strings.Split(fixed, "/")[1:]
	lastNamespace := namespaces[len(namespaces)-1]
	lnparts := strings.Split(lastNamespace, ":")
	ktype := ""
	if len(lnparts) > 1 {
		ktype = strings.Join(lnparts[:len(lnparts)-1], ":")
	}
	kname := lnparts[len(lnparts)-1]

	kchild := path.Clean(fixed + "/cchildd")
	kparent := "/" + strings.Join(append(namespaces[:len(namespaces)-1]), "/")
	kpath := path.Clean(kparent + "/" + ktype)
	kinstance := fixed + ":" + "inst"

	c.Log("Testing: ", NewKey(s))

	c.Check(NewKey(s).String(), Equals, fixed)
	c.Check(NewKey(s), Equals, NewKey(s))
	c.Check(NewKey(s).String(), Equals, NewKey(s).String())
	c.Check(NewKey(s).Name(), Equals, kname)
	c.Check(NewKey(s).Type(), Equals, ktype)
	c.Check(NewKey(s).Path().String(), Equals, kpath)
	c.Check(NewKey(s).Instance("inst").String(), Equals, kinstance)

	c.Check(NewKey(s).Child(NewKey("cchildd")).String(), Equals, kchild)
	c.Check(NewKey(s).Child(NewKey("cchildd")).Parent().String(), Equals, fixed)
	c.Check(NewKey(s).ChildString("cchildd").String(), Equals, kchild)
	c.Check(NewKey(s).ChildString("cchildd").Parent().String(), Equals, fixed)
	c.Check(NewKey(s).Parent().String(), Equals, kparent)
	c.Check(len(NewKey(s).List()), Equals, len(namespaces))
	c.Check(len(NewKey(s).Namespaces()), Equals, len(namespaces))
	for i, e := range NewKey(s).List() {
		c.Check(namespaces[i], Equals, e)
	}

	c.Check(NewKey(s), Equals, NewKey(s))
	c.Check(NewKey(s).Equal(NewKey(s)), Equals, true)
	c.Check(NewKey(s).Equal(NewKey("/fdsafdsa/"+s)), Equals, false)

	// less
	c.Check(NewKey(s).Less(NewKey(s).Parent()), Equals, false)
	c.Check(NewKey(s).Less(NewKey(s).ChildString("foo")), Equals, true)
}

func (ks *KeySuite) TestKeyBasic(c *C) {
	ks.SubtestKey("", c)
	ks.SubtestKey("abcde", c)
	ks.SubtestKey("disahfidsalfhduisaufidsail", c)
	ks.SubtestKey("/fdisahfodisa/fdsa/fdsafdsafdsafdsa/fdsafdsa/", c)
	ks.SubtestKey("4215432143214321432143214321", c)
	ks.SubtestKey("/fdisaha////fdsa////fdsafdsafdsafdsa/fdsafdsa/", c)
	ks.SubtestKey("abcde:fdsfd", c)
	ks.SubtestKey("disahfidsalfhduisaufidsail:fdsa", c)
	ks.SubtestKey("/fdisahfodisa/fdsa/fdsafdsafdsafdsa/fdsafdsa/:", c)
	ks.SubtestKey("4215432143214321432143214321:", c)
	ks.SubtestKey("fdisaha////fdsa////fdsafdsafdsafdsa/fdsafdsa/f:fdaf", c)
}

func CheckTrue(c *C, cond bool) {
	c.Check(cond, Equals, true)
}

func (ks *KeySuite) TestKeyAncestry(c *C) {
	k1 := NewKey("/A/B/C")
	k2 := NewKey("/A/B/C/D")

	c.Check(k1.String(), Equals, "/A/B/C")
	c.Check(k2.String(), Equals, "/A/B/C/D")
	CheckTrue(c, k1.IsAncestorOf(k2))
	CheckTrue(c, k2.IsDescendantOf(k1))
	CheckTrue(c, NewKey("/A").IsAncestorOf(k2))
	CheckTrue(c, NewKey("/A").IsAncestorOf(k1))
	CheckTrue(c, !NewKey("/A").IsDescendantOf(k2))
	CheckTrue(c, !NewKey("/A").IsDescendantOf(k1))
	CheckTrue(c, k2.IsDescendantOf(NewKey("/A")))
	CheckTrue(c, k1.IsDescendantOf(NewKey("/A")))
	CheckTrue(c, !k2.IsAncestorOf(NewKey("/A")))
	CheckTrue(c, !k1.IsAncestorOf(NewKey("/A")))
	CheckTrue(c, !k2.IsAncestorOf(k2))
	CheckTrue(c, !k1.IsAncestorOf(k1))
	c.Check(k1.Child(NewKey("D")).String(), Equals, k2.String())
	c.Check(k1.ChildString("D").String(), Equals, k2.String())
	c.Check(k1.String(), Equals, k2.Parent().String())
	c.Check(k1.Path().String(), Equals, k2.Parent().Path().String())
}

func (ks *KeySuite) TestType(c *C) {
	k1 := NewKey("/A/B/C:c")
	k2 := NewKey("/A/B/C:c/D:d")

	CheckTrue(c, k1.IsAncestorOf(k2))
	CheckTrue(c, k2.IsDescendantOf(k1))
	c.Check(k1.Type(), Equals, "C")
	c.Check(k2.Type(), Equals, "D")
	c.Check(k1.Type(), Equals, k2.Parent().Type())
}

func (ks *KeySuite) TestRandom(c *C) {
	keys := map[Key]bool{}
	for i := 0; i < 1000; i++ {
		r := RandomKey()
		_, found := keys[r]
		CheckTrue(c, !found)
		keys[r] = true
	}
	CheckTrue(c, len(keys) == 1000)
}

func (ks *KeySuite) TestLess(c *C) {

	checkLess := func(a, b string) {
		ak := NewKey(a)
		bk := NewKey(b)
		c.Check(ak.Less(bk), Equals, true)
		c.Check(bk.Less(ak), Equals, false)
	}

	checkLess("/a/b/c", "/a/b/c/d")
	checkLess("/a/b", "/a/b/c/d")
	checkLess("/a", "/a/b/c/d")
	checkLess("/a/a/c", "/a/b/c")
	checkLess("/a/a/d", "/a/b/c")
	checkLess("/a/b/c/d/e/f/g/h", "/b")
	checkLess("/", "/a")
}

func TestKeyMarshalJSON(t *testing.T) {
	cases := []struct {
		key  Key
		data []byte
		err  string
	}{
		{NewKey("/a/b/c"), []byte("\"/a/b/c\""), ""},
		{NewKey("/shouldescapekey\"/with/quote"), []byte("\"/shouldescapekey\\\"/with/quote\""), ""},
	}

	for i, c := range cases {
		out, err := c.key.MarshalJSON()
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d marshal error mismatch: expected: %s, got: %s", i, c.err, err)
		}
		if !bytes.Equal(c.data, out) {
			t.Errorf("case %d value mismatch: expected: %s, got: %s", i, string(c.data), string(out))
		}

		if c.err == "" {
			key := Key{}
			if err := key.UnmarshalJSON(out); err != nil {
				t.Errorf("case %d error parsing key from json output: %s", i, err.Error())
			}
			if !c.key.Equal(key) {
				t.Errorf("case %d parsed key from json output mismatch. expected: %s, got: %s", i, c.key.String(), key.String())
			}
		}
	}
}

func TestKeyUnmarshalJSON(t *testing.T) {
	cases := []struct {
		data []byte
		key  Key
		err  string
	}{
		{[]byte("\"/a/b/c\""), NewKey("/a/b/c"), ""},
		{[]byte{}, Key{}, "unexpected end of JSON input"},
		{[]byte{'"'}, Key{}, "unexpected end of JSON input"},
		{[]byte(`""`), NewKey(""), ""},
	}

	for i, c := range cases {
		key := Key{}
		err := key.UnmarshalJSON(c.data)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d marshal error mismatch: expected: %s, got: %s", i, c.err, err)
		}

		if !key.Equal(c.key) {
			t.Errorf("case %d key mismatch: expected: %s, got: %s", i, c.key, key)
		}
	}
}
