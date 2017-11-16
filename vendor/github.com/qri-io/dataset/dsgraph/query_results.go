package dsgraph

import (
	"encoding/json"
	"github.com/ipfs/go-datastore"
)

// QueryResults graphs query paths to result paths
type QueryResults map[datastore.Key][]datastore.Key

func (qr QueryResults) AddResult(query, result datastore.Key) {
	for _, r := range qr[query] {
		if r.Equal(result) {
			return
		}
	}
	qr[query] = append(qr[query], result)
}

func (qr QueryResults) MarshalJSON() ([]byte, error) {
	qrmap := map[string]interface{}{}
	for key, vals := range qr {
		strs := make([]string, len(vals))
		for i, v := range vals {
			strs[i] = v.String()
		}
		qrmap[key.String()] = strs
	}
	return json.Marshal(qrmap)
}

func (qr *QueryResults) UnmarshalJSON(data []byte) error {
	qrmap := map[string][]datastore.Key{}
	if err := json.Unmarshal(data, &qrmap); err != nil {
		return err
	}

	r := QueryResults{}

	for key, vals := range qrmap {
		r[datastore.NewKey(key)] = vals
	}
	*qr = r
	return nil
}
