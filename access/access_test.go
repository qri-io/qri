package access

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo/profile"
)

func ExamplePolicy() {

	const examplePolicy = `
[
	{
		"title": "pull any dataset",
		"effect": "allow",
		"subject": "*",
		"resources": [
			"dataset:*"
		],
		"actions": [
			"remote:pull"
		]
	},
	{
		"title": "push and delete user-owned datasets",
		"effect": "allow",
		"subject": "*",
		"resources": [
			"dataset:_subject:*"
		],
		"actions": [
			"remote:push",
			"remote:remove"
		]
	}
]
`

	p := &Policy{}
	if err := json.Unmarshal([]byte(examplePolicy), p); err != nil {
		panic(err)
	}

	bob := &profile.Profile{
		ID:       profile.IDB58DecodeOrEmpty("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"),
		Peername: "bob",
	}

	if err := p.Enforce(bob, "dataset:someone_else:world_bank_population", "remote:pull"); err == nil {
		fmt.Println("bob can pull someone_else/world_bank_population")
	}
	if err := p.Enforce(bob, "dataset:bob:bobs_dataset", "remote:remove"); err == nil {
		fmt.Println("bob can remote-delete his own dataset")
	}
	if err := p.Enforce(bob, "dataset:someone_else:dataset", "remote:remove"); err == ErrAccessDenied {
		fmt.Println("bob can't remote-delete someone else's dataset")
	}

	// Output:
	// bob can pull someone_else/world_bank_population
	// bob can remote-delete his own dataset
	// bob can't remote-delete someone else's dataset
}

func TestEnforce(t *testing.T) {
	pro := &profile.Profile{
		ID:       profile.IDB58DecodeOrEmpty("QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt"),
		Peername: "bob",
	}

	p := Policy{
		{
			Subject:   "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
			Resources: Resources{MustParseResource("dataset:foo:bar")},
			Actions:   Actions{MustParseAction("*")},
			Effect:    EffectAllow,
		},
	}

	if err := p.Enforce(pro, "dataset:foo:bar", "read"); err != nil {
		t.Error(err)
	}
}

func TestPolicyJSON(t *testing.T) {
	bad := [][2]string{
		{"rule.Subject is required", `[{}]`},
		{"rule.Resources field is required", `[{
			"subject": "*", 
			"resources": [], 
			"actions": ["*"],
			"effect": "deny"
		}]`},
		{`rule.Effect must be one of ("allow"|"deny")`, `[{
			"subject": "*", 
			"resources": ["*"], 
			"actions": ["*"],
			"effect": "evaporate"
		}]`},
		{`rule.Actions field is required`, `[{
			"subject": "*", 
			"resources": ["*"], 
			"action": ["*"],
			"effect": "allow"
		}]`},
	}

	for _, c := range bad {
		t.Run(c[0], func(t *testing.T) {
			pol := &Policy{}

			err := json.Unmarshal([]byte(c[1]), pol)
			if err == nil {
				t.Fatal("expected bad policy to fail. received no error")
			}

			if err.Error() != c[0] {
				t.Errorf("error message mismatch. want: %q\ngot:  %q", c[1], err.Error())
			}
		})
	}
}

func TestParseResource(t *testing.T) {
	bad := []string{
		"",
		"*:foo",
	}

	for _, str := range bad {
		if _, err := ParseResource(str); err == nil {
			t.Errorf("expected error parsing bad resource. got nil.")
		}
	}
}

func TestResourceContains(t *testing.T) {
	cases := []struct {
		a, b   string
		expect bool
	}{
		{"*", "apples", true},
		{"candy:*", "apples", false},
		{"candy:apples", "candy:apples", true},
		{"candy:apples", "candy:applez", false},
		{"dataset:foo:bar", "dataset", false},
		{"dataset:foo:bar", "dataset:foo:bar:baz", false},
		{"dataset:foo:*", "dataset:foo:bar:baz", true},
		{"dataset:foo:bar:*", "dataset:foo:bar:baz", true},

		{"dataset:*", "dataset:someone_else:world_bank_population", true},

		{"dataset:_subject:bar:*", "dataset:user:bar:baz", true},
		{"dataset:_subject:bar:*", "dataset:other_user:bar:baz", false},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s_contains_%s_is_%t", c.a, c.b, c.expect), func(t *testing.T) {
			rscA, err := ParseResource(c.a)
			if err != nil {
				t.Fatal(err)
			}

			rscB, err := ParseResource(c.b)
			if err != nil {
				t.Fatal(err)
			}

			got := rscA.Contains(rscB, "user")
			if c.expect != got {
				t.Errorf("result mismatch expected %q.Contains(%q) == %t", c.a, c.b, c.expect)
			}

		})
	}
}

func TestActionContains(t *testing.T) {
	cases := []struct {
		a, b   string
		expect bool
	}{
		{"*", "apples", true},
		{"candy:*", "apples", false},
		{"candy:apples", "candy:apples", true},
		{"candy:apples", "candy:applez", false},
		{"dataset:foo:bar", "dataset", false},
		{"dataset:foo:bar", "dataset:foo:bar:baz", false},
		{"dataset:foo:*", "dataset:foo:bar:baz", true},
		{"dataset:foo:bar:*", "dataset:foo:bar:baz", true},

		{"remote:pull", "remote:pull", true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s_contains_%s_is_%t", c.a, c.b, c.expect), func(t *testing.T) {
			actA := MustParseAction(c.a)
			actB := MustParseAction(c.b)

			got := actA.Contains(actB)
			if c.expect != got {
				t.Errorf("result mismatch expected %q.Contains(%q) == %t", c.a, c.b, c.expect)
			}
		})
	}
}

func TestResourceStrFromRef(t *testing.T) {
	cases := []struct {
		ref    dsref.Ref
		expect string
	}{
		{dsref.Ref{Username: "foo", Name: "bar"}, "dataset:foo:bar"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("ref_%s_becomes_resource_%s", c.ref, c.expect), func(t *testing.T) {

			got := ResourceStrFromRef(c.ref)
			if c.expect != got {
				t.Errorf("result mismatch expected ResourceStrFromRef(%s) == %s", c.ref, c.expect)
			}
		})
	}
}
