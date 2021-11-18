package staticlark

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestUnitCases(t *testing.T) {
	cases := []struct {
		desc string
		unit unit
		str  string
		srcs []string
		invs string
	}{
		{
			"atomic identifer",
			unit{
				atom: "a",
			},
			"a",
			[]string{},
			`[]`,
		},
		{
			"assign literal to var",
			unit{
				atom: "set!",
				tail: []*unit{
					{atom: "a"},
					{atom: "0"},
				},
			},
			"[set! a 0]",
			[]string{},
			`[]`,
		},
		{
			"var assigned another var",
			unit{
				atom: "set!",
				tail: []*unit{
					{atom: "a"},
					{atom: "b"},
				},
			},
			"[set! a b]",
			[]string{"b"},
			`[]`,
		},
		{
			"increment a var",
			unit{
				atom: "set!",
				tail: []*unit{
					{atom: "a"},
					{
						atom: "+=",
						tail: []*unit{
							{atom: "a"},
							{atom: "1"},
						},
					},
				},
			},
			"[set! a [+= a 1]]",
			[]string{"a"},
			`[]`,
		},
		{
			"add two vars and assign",
			unit{
				atom: "set!",
				tail: []*unit{
					{atom: "a"},
					{
						atom: "+",
						tail: []*unit{
							{atom: "a"},
							{atom: "b"},
						},
					},
				},
			},
			"[set! a [+ a b]]",
			[]string{"a", "b"},
			`[]`,
		},
		{
			"assign var from function call",
			unit{
				atom: "set!",
				tail: []*unit{
					{atom: "a"},
					{
						atom: "add",
						tail: []*unit{
							{atom: "b"},
							{atom: "c"},
						},
					},
				},
			},
			"[set! a [add b c]]",
			[]string{"b", "c"},
			`[{"name":"add","args":["b","c"]}]`,
		},
		{
			"just function call",
			unit{
				atom: "upload",
				tail: []*unit{
					{atom: "a"},
					{atom: "b"},
					{atom: "c"},
				},
			},
			"[upload a b c]",
			[]string{"b", "c"},
			`[{"name":"upload","args":["a","b","c"]}]`,
		},
		{
			"nested function calls",
			unit{
				atom: "upload",
				tail: []*unit{
					{atom: "a"},
					{atom: "b"},
					{
						atom: "lookup",
						tail: []*unit{
							{atom: "a"},
							{atom: "c"},
						},
					},
				},
			},
			"[upload a b [lookup a c]]",
			[]string{"b", "a", "c"},
			`[{"name":"upload","args":["a","b","a","c"]},{"name":"lookup","args":["a","c"]}]`,
		},
		{
			"return a value",
			unit{
				atom: "return",
				tail: []*unit{
					{atom: "a"},
				},
			},
			"[return a]",
			[]string{},
			`[]`,
		},
		{
			"return an expression",
			unit{
				atom: "return",
				tail: []*unit{
					{
						atom: "add",
						tail: []*unit{
							{atom: "a"},
							{atom: "b"},
						},
					},
				},
			},
			"[return [add a b]]",
			// TODO(dustmop): Is this correct?
			[]string{},
			`[{"name":"add","args":["a","b"]}]`,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if diff := cmp.Diff(c.str, c.unit.String()); diff != "" {
				t.Errorf("String(), mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(c.srcs, c.unit.DataSources()); diff != "" {
				t.Errorf("DataSources(), mismatch (-want +got):\n%s", diff)
			}
			invs := c.unit.Invocations()
			data, err := json.Marshal(invs)
			if err != nil {
				t.Fatal(err)
			}
			actual := string(data)
			if diff := cmp.Diff(c.invs, actual); diff != "" {
				t.Errorf("Invocations(), mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
