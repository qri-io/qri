package p2p

import (
	"github.com/qri-io/qri/repo"
	// pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
)

// MtSearch is a search message
const MtSearch = MsgType("SEARCH")

// Search broadcasts a search request to all connected peers, aggregating results
func (n *QriNode) Search(terms string, limit, offset int) (res []*repo.DatasetRef, err error) {
	// responses, err := n.BroadcastMessage(&Message{
	// 	Phase: MpRequest,
	// 	Type:  MtSearch,
	// 	Payload: &repo.SearchParams{
	// 		Q:      terms,
	// 		Limit:  limit,
	// 		Offset: offset,
	// 	},
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// datasets := []*repo.DatasetRef{}

	// for _, r := range responses {
	// 	data, err := json.Marshal(r.Payload)
	// 	if err != nil {
	// 		return datasets, err
	// 	}
	// 	ds := []*repo.DatasetRef{}
	// 	if err := json.Unmarshal(data, &ds); err != nil {
	// 		return datasets, err
	// 	}
	// 	datasets = append(datasets, ds...)
	// }

	return res, nil
}

func (n *QriNode) handleSearchRequest(ws *WrappedStream, msg Message) (hangup bool) {
	// log.Debug("handling search request")
	// data, err := json.Marshal(msg.Payload)
	// if err != nil {
	// 	log.Info(err.Error())
	// 	return true
	// }
	// p := &repo.SearchParams{}
	// if err := json.Unmarshal(data, p); err != nil {
	// 	log.Info("unmarshal search request error:", err.Error())
	// 	return true
	// }

	// // results, err := search.Search(n.Repo, n.Store, search.NewDatasetQuery(p.Query, p.Limit, p.Offset))
	// if s, ok := n.Repo.(repo.Searchable); ok {
	// 	results, err := s.Search(*p)
	// 	if err != nil {
	// 		log.Info("search error:", err.Error())
	// 		return nil
	// 	}
	// 	return &Message{
	// 		Phase:   MpResponse,
	// 		Type:    MtSearch,
	// 		Payload: results,
	// 	}
	// }

	// &Message{
	//     Phase:   MpError,
	//     Type:    MtSearch,
	//     Payload: fmt.Errorf("repo doesn't support search"),
	//   }
	return
}
