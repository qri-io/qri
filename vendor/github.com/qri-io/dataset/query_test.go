package dataset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"

	"testing"
)

func TestQueryAssign(t *testing.T) {
	expect := &Query{
		path:       datastore.NewKey("path"),
		Syntax:     "a",
		AppVersion: "b",
		Config: map[string]interface{}{
			"foo": "bar",
		},
		Abstract: &AbstractQuery{
			Syntax: "abstract_syntax",
		},
		Resources: map[string]*Dataset{
			"a": NewDatasetRef(datastore.NewKey("/path/to/a")),
		},
	}
	got := &Query{
		Syntax:     "no",
		AppVersion: "change",
		Config: map[string]interface{}{
			"foo": "baz",
		},
		Abstract:  nil,
		Resources: nil,
	}

	got.Assign(&Query{
		Syntax:     "a",
		AppVersion: "b",
		Config: map[string]interface{}{
			"foo": "bar",
		},
		Abstract:  nil,
		Resources: nil,
	}, &Query{
		Abstract: &AbstractQuery{
			Syntax: "abstract_syntax",
		},
		Resources: map[string]*Dataset{
			"a": NewDatasetRef(datastore.NewKey("/path/to/a")),
		},
	})

	if err := CompareQuery(expect, got); err != nil {
		t.Error(err)
	}

	got.Assign(nil, nil)
	if err := CompareQuery(expect, got); err != nil {
		t.Error(err)
	}

	emptyMsg := &Query{}
	emptyMsg.Assign(expect)
	if err := CompareQuery(expect, emptyMsg); err != nil {
		t.Error(err)
	}
}

func CompareQuery(a, b *Query) error {
	if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("nil mismatch: %v != %v", a, b)
	}
	if a == nil && b == nil {
		return nil
	}
	if err := CompareAbstractQuery(a.Abstract, b.Abstract); err != nil {
		return err
	}
	if len(a.Resources) != len(b.Resources) {
		return fmt.Errorf("resource count mistmatch: %d != %d", len(a.Resources), len(b.Resources))
	}
	for key, val := range a.Resources {
		if err := CompareDatasets(val, b.Resources[key]); err != nil {
			return err
		}
	}
	return nil
}

func TestQueryUnmarshalJSON(t *testing.T) {
	cases := []struct {
		str   string
		query *Query
		err   error
	}{
		{`{}`, &Query{}, nil},
		{`{ "abstract" : "/path/to/abstract" }`, &Query{Abstract: &AbstractQuery{path: datastore.NewKey("/path/to/abstract")}}, nil},
		// {`{ "syntax" : "ql", "statement" : "select a from b" }`, &Query{Syntax: "ql", Statement: "select a from b"}, nil},
	}

	for i, c := range cases {
		got := &Query{}
		if err := json.Unmarshal([]byte(c.str), got); err != c.err {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}

		if err := CompareQuery(c.query, got); err != nil {
			t.Errorf("case %d query mismatch: %s", i, err)
			continue
		}
	}

	strq := &Query{}
	path := "/path/to/query"
	if err := json.Unmarshal([]byte(`"`+path+`"`), strq); err != nil {
		t.Errorf("unmarshal string path error: %s", err.Error())
		return
	}

	if strq.path.String() != path {
		t.Errorf("unmarshal didn't set proper path: %s != %s", path, strq.path)
		return
	}
}

func TestQueryMarshalJSON(t *testing.T) {
	cases := []struct {
		q   *Query
		out string
		err error
	}{
		{&Query{}, `{}`, nil},
		// {&Query{Syntax: "sql", Statement: "select a from b"}, `{"outputStructure":null,"statement":"select a from b","structures":null,"syntax":"sql"}`, nil},
	}

	for i, c := range cases {
		data, err := json.Marshal(c.q)
		if err != c.err {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		if string(data) != c.out {
			t.Errorf("case %d result mismatch. expected: %s, got: %s", i, c.out, string(data))
			continue
		}
	}

	strbytes, err := json.Marshal(&Query{path: datastore.NewKey("/path/to/query")})
	if err != nil {
		t.Errorf("unexpected string marshal error: %s", err.Error())
		return
	}

	if !bytes.Equal(strbytes, []byte("\"/path/to/query\"")) {
		t.Errorf("marshal strbyte interface byte mismatch: %s != %s", string(strbytes), "\"/path/to/query\"")
	}
}

func CompareAbstractQuery(a, b *AbstractQuery) error {
	if a == nil && b != nil || a != nil && b == nil {
		return fmt.Errorf("nil mismatch: %v != %v", a, b)
	}
	if a == nil && b == nil {
		return nil
	}
	if a.Syntax != b.Syntax {
		return fmt.Errorf("syntax mismatch: %s != %s", a.Syntax, b.Syntax)
	}
	if a.Statement != b.Statement {
		return fmt.Errorf("statement mismatch: %s != %s", a.Statement, b.Statement)
	}

	return nil
}

func TestAbstractQueryAssign(t *testing.T) {
	expect := &AbstractQuery{
		path:      datastore.NewKey("/path/to/abstract/query"),
		Statement: "what a statement",
		Structure: &Structure{
			Schema: &Schema{
				Fields: []*Field{
					&Field{Name: "col_a"},
					&Field{Name: "col_b"},
				},
			},
		},
		Structures: map[string]*Structure{
			"a": &Structure{
				Format: CsvDataFormat,
			},
		},
		Syntax: "foobar",
	}
	got := &AbstractQuery{
		path:      datastore.NewKey("/clobber/me/plz"),
		Statement: "who the statement",
	}

	got.Assign(&AbstractQuery{
		path:      datastore.NewKey("/path/to/abstract/query"),
		Statement: "what a statement",
		Structure: &Structure{
			Schema: &Schema{
				Fields: []*Field{
					&Field{Name: "col_a"},
					&Field{Name: "col_b"},
				},
			},
		},
	}, &AbstractQuery{
		Structures: map[string]*Structure{
			"a": &Structure{
				Format: CsvDataFormat,
			},
		},
		Syntax: "foobar",
	})

	if err := CompareAbstractQuery(expect, got); err != nil {
		t.Error(err)
	}

	got.Assign(nil, nil)
	if err := CompareAbstractQuery(expect, got); err != nil {
		t.Error(err)
	}

	emptyMsg := &AbstractQuery{}
	emptyMsg.Assign(expect)
	if err := CompareAbstractQuery(expect, emptyMsg); err != nil {
		t.Error(err)
	}
}

func TestAbstractQueryUnmarshalJSON(t *testing.T) {
	cases := []struct {
		str   string
		query *AbstractQuery
		err   error
	}{
		{`{ "statement" : "select a from b" }`, &AbstractQuery{Statement: "select a from b"}, nil},
		{`{ "syntax" : "ql", "statement" : "select a from b" }`, &AbstractQuery{Syntax: "ql", Statement: "select a from b"}, nil},
	}

	for i, c := range cases {
		got := &AbstractQuery{}
		if err := json.Unmarshal([]byte(c.str), got); err != c.err {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}

		if err := CompareAbstractQuery(c.query, got); err != nil {
			t.Errorf("case %d query mismatch: %s", i, err)
			continue
		}
	}

	strq := &AbstractQuery{}
	path := "/path/to/query"
	if err := json.Unmarshal([]byte(`"`+path+`"`), strq); err != nil {
		t.Errorf("unmarshal string path error: %s", err.Error())
		return
	}

	if strq.path.String() != path {
		t.Errorf("unmarshal didn't set proper path: %s != %s", path, strq.path)
		return
	}
}

func TestAbstractQueryMarshalJSON(t *testing.T) {
	cases := []struct {
		q   *AbstractQuery
		out string
		err error
	}{
		{&AbstractQuery{Syntax: "sql", Statement: "select a from b"}, `{"outputStructure":null,"statement":"select a from b","structures":null,"syntax":"sql"}`, nil},
	}

	for i, c := range cases {
		data, err := json.Marshal(c.q)
		if err != c.err {
			t.Errorf("case %d error mismatch. expected: %s, got: %s", i, c.err, err)
			continue
		}
		if string(data) != c.out {
			t.Errorf("case %d result mismatch. expected: %s, got: %s", i, c.out, string(data))
			continue
		}
	}

	strbytes, err := json.Marshal(&AbstractQuery{path: datastore.NewKey("/path/to/abstractquery")})
	if err != nil {
		t.Errorf("unexpected string marshal error: %s", err.Error())
		return
	}

	if !bytes.Equal(strbytes, []byte("\"/path/to/abstractquery\"")) {
		t.Errorf("marshal strbyte interface byte mismatch: %s != %s", string(strbytes), "\"/path/to/abstractquery\"")
	}
}
