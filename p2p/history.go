package p2p

import (
	"github.com/qri-io/qri/repo"
)

const (
	// MtDatasetLog gets log of a dataset
	MtDatasetLog = MsgType("dataset_history")
)

// RequestDatasetLog gets the log information of Peer's dataset
func (n *QriNode) RequestDatasetLog(ref repo.DatasetRef) (*[]repo.DatasetRef, error) {
	// id, err := n.Repo.Peers().IPFSPeerID(ref.Peername)
	// if err != nil {
	// 	return nil, fmt.Errorf("error getting peer IPFS id: %s", err.Error())
	// }
	// res, err := n.SendMessage(id, &Message{
	// 	Type:    MtDatasetLog,
	// 	Phase:   MpRequest,
	// 	Payload: ref,
	// })
	// if err != nil {
	// 	log.Debugf("send dataset log message error: %s", err.Error())
	// 	return nil, err
	// }

	// data, err := json.Marshal(res.Payload)
	// if err != nil {
	// 	log.Debug(err.Error())
	// 	return nil, err
	// }

	resref := []repo.DatasetRef{}
	// err = json.Unmarshal(data, &resref)
	// if len(resref) == 0 && err != nil {
	// 	err = fmt.Errorf("no log found")
	// }

	return &resref, nil
}

func (n *QriNode) datasetsHistoryHandler(ws *WrappedStream, msg Message) (hangup bool) {
	// data, err := json.Marshal(msg.Payload)
	// if err != nil {
	// 	log.Debug(err.Error())
	// }

	return false

	// ref := repo.DatasetRef{}
	// if err = json.Unmarshal(data, &ref); err != nil {
	// 	log.Infof(err.Error())
	// 	return &Message{
	// 		Type:    MtDatasetLog,
	// 		Phase:   MpError,
	// 		Payload: err,
	// 	}
	// }

	// ref, err = n.Repo.GetRef(ref)
	// if err != nil {
	// 	return &Message{
	// 		Type:    MtDatasetLog,
	// 		Phase:   MpError,
	// 		Payload: err,
	// 	}
	// }
	// // TODO: probably shouldn't write over ref.Path if ref.Path is set, but
	// // until we make the changes to the way we use hashes to make them
	// // more consistent, this feels safer.
	// // ref.Path = path.String()

	// log := []repo.DatasetRef{}
	// limit := 50

	// for {
	// 	ref.Dataset, err = n.Repo.GetDataset(datastore.NewKey(ref.Path))
	// 	if err != nil {
	// 		return &Message{
	// 			Type:    MtDatasetLog,
	// 			Phase:   MpError,
	// 			Payload: err,
	// 		}
	// 	}
	// 	log = append(log, ref)

	// 	limit--
	// 	if limit == 0 || ref.Dataset.PreviousPath == "" {
	// 		break
	// 	}

	// 	ref, err = repo.ParseDatasetRef(ref.Dataset.PreviousPath)

	// 	if err != nil {
	// 		return &Message{
	// 			Type:    MtDatasetLog,
	// 			Phase:   MpError,
	// 			Payload: err,
	// 		}
	// 	}
	// }
	// return &Message{
	// 	Type:    MtDatasetLog,
	// 	Phase:   MpResponse,
	// 	Payload: &log,
	// }
}
