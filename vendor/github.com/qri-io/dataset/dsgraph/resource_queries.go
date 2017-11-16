package dsgraph

// import (
// 	"encoding/json"
// 	"github.com/ipfs/go-datastore"
// )

// // ResourceQueries connects a resource path to queries
// // that consume the resource
// type ResourceQueries map[datastore.Key][]datastore.Key

// func (qr ResourceQueries) AddQuery(resource, query datastore.Key) {
// 	for _, r := range qr[resource] {
// 		if r.Equal(query) {
// 			return
// 		}
// 	}
// 	qr[resource] = append(qr[resource], query)
// }

// func (qr ResourceQueries) MarshalJSON() ([]byte, error) {
// 	strmap := map[string]interface{}{}
// 	for key, vals := range qr {
// 		strs := make([]string, len(vals))
// 		for i, v := range vals {
// 			strs[i] = v.String()
// 		}
// 		strmap[key.String()] = strs
// 	}
// 	return json.Marshal(strmap)
// }

// func (qr *ResourceQueries) UnmarshalJSON(data []byte) error {
// 	strmap := map[string][]datastore.Key{}
// 	if err := json.Unmarshal(data, &strmap); err != nil {
// 		return err
// 	}

// 	r := ResourceQueries{}

// 	for key, vals := range strmap {
// 		r[datastore.NewKey(key)] = vals
// 	}
// 	*qr = r
// 	return nil
// }
