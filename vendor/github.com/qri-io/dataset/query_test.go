package dataset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ipfs/go-datastore"

	"testing"
)

func CompareQuery(a, b *Query) error {
	if a.Syntax != b.Syntax {
		return fmt.Errorf("syntax mismatch: %s != %s", a.Syntax, b.Syntax)
	}
	if a.Statement != b.Statement {
		return fmt.Errorf("statement mismatch: %s != %s", a.Statement, b.Statement)
	}

	return nil
}

func TestQueryUnmarshalJSON(t *testing.T) {
	cases := []struct {
		str   string
		query *Query
		err   error
	}{
		// This no long works, place taken by path unmarshaling
		// {`"select a from b"`, &Query{Statement: "select a from b"}, nil},
		{`{ "statement" : "select a from b" }`, &Query{Statement: "select a from b"}, nil},
		{`{ "syntax" : "ql", "statement" : "select a from b" }`, &Query{Syntax: "ql", Statement: "select a from b"}, nil},
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
		{&Query{Syntax: "sql", Statement: "select a from b"}, `{"outputStructure":null,"statement":"select a from b","structures":null,"syntax":"sql"}`, nil},
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

	strbytes, err := json.Marshal(&Query{path: datastore.NewKey("/path/to/dataset")})
	if err != nil {
		t.Errorf("unexpected string marshal error: %s", err.Error())
		return
	}

	if !bytes.Equal(strbytes, []byte("\"/path/to/dataset\"")) {
		t.Errorf("marshal strbyte interface byte mismatch: %s != %s", string(strbytes), "\"/path/to/dataset\"")
	}
}
