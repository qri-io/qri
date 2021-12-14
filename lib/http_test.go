package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/base/params"
)

func TestDecodeParams(t *testing.T) {
	p := &CollectionListParams{}
	r, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	expect := &CollectionListParams{
		List: params.List{
			Offset:  10,
			Limit:   10,
			Filter:  []string{"username:peer"},
			OrderBy: params.NewOrderByFromString("+name,-updated"),
		},
	}
	// add list queries to the request
	q := r.URL.Query()
	q.Add("offset", "10")
	q.Add("limit", "10")
	q.Add("filter", "username:peer")
	q.Add("orderby", "+name,-updated")
	r.URL.RawQuery = q.Encode()

	if err := DecodeParams(r, p); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expect, p); diff != "" {
		t.Errorf("CollectionListParams mismatch (+want,-got):\n%s", diff)
	}

	lp := params.List{}
	if err := DecodeParams(r, &lp); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expect.List, lp); diff != "" {
		t.Errorf("list.Params mismatch (+want,-got):\n%s", diff)
	}

	// Decode params should not error if a param is not a `params.ListParams`
	cgp := &CollectionGetParams{}
	expectCGP := &CollectionGetParams{}
	if err := DecodeParams(r, &cgp); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectCGP, cgp); diff != "" {
		t.Errorf("list.Params mismatch (+want,-got):\n%s", diff)
	}

	enc, err := json.Marshal(CollectionListParams{List: params.List{Offset: 10}})
	if err != nil {
		t.Fatal(err)
	}

	r, err = http.NewRequest("POST", "/test", bytes.NewReader(enc))
	if err != nil {
		t.Fatal(err)
	}

	// add list queries to the request
	q = r.URL.Query()
	q.Add("offset", "10")
	q.Add("limit", "10")
	q.Add("filter", "username:peer")
	q.Add("orderby", "+name,-updated")
	r.URL.RawQuery = q.Encode()

	cp := &CollectionListParams{}
	if err := DecodeParams(r, cp); !errors.Is(err, params.ErrListParamsNotEmpty) {
		t.Fatalf("error mismatch, expected %q, got %q", params.ErrListParamsNotEmpty, err)
	}
}
