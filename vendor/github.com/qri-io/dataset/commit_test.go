package dataset

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/ipfs/go-datastore"
)

func TestCommitMsgAssign(t *testing.T) {
	doug := &User{Id: "doug_id", Email: "doug@example.com"}
	expect := &CommitMsg{
		Author:  doug,
		Title:   "expect title",
		Message: "expect message",
	}
	got := &CommitMsg{
		Author:  &User{Id: "maha_id", Email: "maha@example.com"},
		Title:   "title",
		Message: "message",
	}

	got.Assign(&CommitMsg{
		Author: doug,
		Title:  "expect title",
	}, &CommitMsg{
		Message: "expect message",
	})

	if err := CompareCommitMsgs(expect, got); err != nil {
		t.Error(err)
	}

	got.Assign(nil, nil)
	if err := CompareCommitMsgs(expect, got); err != nil {
		t.Error(err)
	}

	emptyMsg := &CommitMsg{}
	emptyMsg.Assign(expect)
	if err := CompareCommitMsgs(expect, emptyMsg); err != nil {
		t.Error(err)
	}
}

func TestCommitMsgMarshalJSON(t *testing.T) {
	cases := []struct {
		in  *CommitMsg
		out []byte
		err error
	}{
		{&CommitMsg{Title: "title"}, []byte(`{"title":"title"}`), nil},
		{&CommitMsg{Author: &User{Id: "foo"}}, []byte(`{"author":{"id":"foo"},"title":""}`), nil},
	}

	for i, c := range cases {
		got, err := c.in.MarshalJSON()
		if err != c.err {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if !bytes.Equal(c.out, got) {
			t.Errorf("case %d error mismatch. %s != %s", i, string(c.out), string(got))
			continue
		}
	}

	strbytes, err := json.Marshal(&CommitMsg{path: datastore.NewKey("/path/to/dataset")})
	if err != nil {
		t.Errorf("unexpected string marshal error: %s", err.Error())
		return
	}

	if !bytes.Equal(strbytes, []byte("\"/path/to/dataset\"")) {
		t.Errorf("marshal strbyte interface byte mismatch: %s != %s", string(strbytes), "\"/path/to/dataset\"")
	}
}

func TestCommitMsgUnmarshalJSON(t *testing.T) {
	cases := []struct {
		data   string
		result *CommitMsg
		err    error
	}{
		{`{}`, &CommitMsg{}, nil},
		{`{ "title": "title", "message": "message"}`, &CommitMsg{Title: "title", Message: "message"}, nil},
		{`{ "author" : { "id": "id", "email": "email@email.com"} }`, &CommitMsg{Author: &User{Id: "id", Email: "email@email.com"}}, nil},
	}

	for i, c := range cases {
		cm := &CommitMsg{}
		if err := json.Unmarshal([]byte(c.data), cm); err != c.err {
			t.Errorf("case %d error mismatch. expected: '%s', got: '%s'", i, c.err, err)
			continue
		}

		if err := CompareCommitMsgs(cm, c.result); err != nil {
			t.Errorf("case %d comparison error: %s", i, err)
			continue
		}
	}

	strq := &CommitMsg{}
	path := "/path/to/msg"
	if err := json.Unmarshal([]byte(`"`+path+`"`), strq); err != nil {
		t.Errorf("unmarshal string path error: %s", err.Error())
		return
	}

	if strq.path.String() != path {
		t.Errorf("unmarshal didn't set proper path: %s != %s", path, strq.path)
		return
	}
}
