package dsgraph

// import (
// 	"encoding/json"
// 	"github.com/ipfs/go-datastore"
// )

// // ResourceMeta adds metadata to a resource key
// type ResourceMeta map[datastore.Key]datastore.Key

// func (rm ResourceMeta) SetMeta(resource, meta datastore.Key) {
// 	rm[resource] = meta
// }

// func (rm ResourceMeta) MarshalJSON() ([]byte, error) {
// 	rmmap := map[string]interface{}{}
// 	for key, val := range rm {
// 		rmmap[key.String()] = val.String()
// 	}
// 	return json.Marshal(rmmap)
// }

// func (rm *ResourceMeta) UnmarshalJSON(data []byte) error {
// 	rmmap := map[string]datastore.Key{}
// 	if err := json.Unmarshal(data, &rmmap); err != nil {
// 		return err
// 	}

// 	r := ResourceMeta{}
// 	for key, val := range rmmap {
// 		r[datastore.NewKey(key)] = val
// 	}
// 	*rm = r
// 	return nil
// }
