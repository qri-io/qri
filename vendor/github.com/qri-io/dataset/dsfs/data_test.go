package dsfs

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestLoadData(t *testing.T) {
	datasets, store, err := makeFilestore()
	if err != nil {
		t.Errorf("error creating test filestore: %s", err.Error())
		return
	}

	ds, err := LoadDataset(store, datasets["movies"])
	if err != nil {
		t.Errorf("error loading dataset: %s", err.Error())
		return
	}

	f, err := LoadData(store, ds)
	if err != nil {
		t.Errorf("error loading data: %s", err.Error())
		return
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Errorf("error reading data file: %s", err.Error())
		return
	}

	eq, err := ioutil.ReadFile("testdata/movies.csv")
	if err != nil {
		t.Errorf("error reading test file: %s", err.Error())
		return
	}

	if !bytes.Equal(data, eq) {
		t.Errorf("byte mismatch. expected: %s, got: %s", string(eq), string(data))
	}
}

func TestLoadRows(t *testing.T) {
	datasets, store, err := makeFilestore()
	if err != nil {
		t.Errorf("error creating test filestore: %s", err.Error())
		return
	}

	cases := []struct {
		dsname        string
		limit, offset int
		expect        string
		err           string
	}{
		// {"cities", 0, 0, "", ""},
		{"cities", 2, 2, `chicago,300000,44.4,true
chatham,35000,65.25,true
`, ""},
		{"archive", 10, 0, `!OpenWayback-CDXJ 1.0
(com,cnn,)/world> 2015-09-03T13:27:52Z response {"a":0,"b":"b","c":false}
(uk,ac,rpms,)/> 2015-09-03T13:27:52Z request {"frequency":241,"spread":3}
(uk,co,bbc,)/images> 2015-09-03T13:27:52Z response {"frequency":725,"spread":1}
`, ""},
	}

	for i, c := range cases {
		ds, err := LoadDataset(store, datasets[c.dsname])
		if err != nil {
			t.Errorf("case %d error loading dataset: %s", i, err.Error())
			continue
		}

		data, err := LoadRows(store, ds, c.limit, c.offset)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case %d unexpected error: %s", i, err.Error())
			continue
		}

		if !bytes.Equal([]byte(c.expect), data) {
			t.Errorf("case %d data mismatch. expected: %s, got: %s", i, c.expect, string(data))
			continue
		}
	}
}
