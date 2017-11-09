package dsio

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/qri-io/dataset"
)

type testCase struct {
	ds   *dataset.Dataset
	data []byte
	rows [][][]byte
}

var rows = map[string][][][]byte{
	"cities": [][][]byte{
		[][]byte{[]byte("toronto"), []byte("40000000"), []byte("55.5"), []byte("false")},
		[][]byte{[]byte("new york"), []byte("8500000"), []byte("44.4"), []byte("true")},
		[][]byte{[]byte("chicago"), []byte("300000"), []byte("44.4"), []byte("true")},
		[][]byte{[]byte("chatham"), []byte("35000"), []byte("65.25"), []byte("true")},
		[][]byte{[]byte("raleigh"), []byte("250000"), []byte("50.65"), []byte("true")},
	},
}

func makeTestData() (map[string]*testCase, error) {
	datasets := map[string]*testCase{
		"movies": nil,
		"cities": nil,
	}

	for k, _ := range datasets {
		data, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s.csv", k))
		if err != nil {
			return datasets, err
		}
		dsdata, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s.json", k))
		if err != nil {
			return datasets, err
		}

		ds := &dataset.Dataset{}
		if err := json.Unmarshal(dsdata, ds); err != nil {
			return datasets, err
		}

		datasets[k] = &testCase{
			ds:   ds,
			data: data,
			rows: rows[k],
		}
	}

	return datasets, nil
}
