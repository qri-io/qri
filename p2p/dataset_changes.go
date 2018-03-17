package p2p

import (
	"encoding/json"
	"strings"
	// "time"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	ipfs "github.com/qri-io/cafs/ipfs"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

// MtDatasetChanges is a message to announce added / removed datasets to the network
const MtDatasetChanges = MsgType("dataset_changes")

// DatasetChanges describes created & deleted datasets with slices of
// repo DatsetRef strings.
// Because dataset data is immutable, All changes should be describable
// as creations and deletions.
// Dataset names, however, *are* mutable. Renaming is conveyed by listing
// the former ref as deleted & new ref as created.
type DatasetChanges struct {
	Created []string
	Deleted []string
}

// AnnounceDatasetChanges transmits info of dataset changes to
func (n *QriNode) AnnounceDatasetChanges(changes DatasetChanges) error {
	log.Debugf("%s: AnnounceDatasetChanges", n.ID)

	msg, err := NewJSONBodyMessage(n.ID, MtDatasetChanges, changes)
	if err != nil {
		return err
	}

	// grab 50 peers & fire off our announcement to them
	return n.SendMessage(msg, nil, n.ClosestConnectedPeers("", 50)...)
}

func (n *QriNode) handleDatasetChanges(ws *WrappedStream, msg Message) (hangup bool) {
	hangup = true

	// only handle messages once, and if they're not too old
	// if _, ok := n.msgState.Load(msg.ID); ok || time.Now().After(msg.Deadline) {
	// 	return
	// }

	n.msgState.Store(msg.ID, msg.Deadline)

	pro, err := n.Repo.Profile()
	if err != nil {
		log.Debug(err.Error())
		return
	}

	changes := DatasetChanges{}
	if err := json.Unmarshal(msg.Body, &changes); err != nil {
		log.Debug(err.Error())
		return
	}

	for _, add := range changes.Created {
		if ref, err := repo.ParseDatasetRef(add); err == nil {
			if err := n.Repo.RefCache().PutRef(ref); err != nil {
				log.Debug(err.Error())
			}

			if ref.PeerID == pro.ID && n.selfReplication == "full" {
				log.Infof("%s %s self replicating dataset %s from %s", n.ID.Pretty(), pro.ID, ref.String(), msg.Initiator.Pretty())

				if err = n.Repo.PutRef(ref); err != nil {
					log.Debug(err.Error())
					return
				}

				go func() {
					// TODO - this is pulled out of core.DatasetRequests.Add
					// we should find a common place for this code & have both add & this func call it
					// possibly in / subpackage of repo?
					fs, ok := n.Repo.Store().(*ipfs.Filestore)
					if !ok {
						log.Debug("can only add datasets when running an IPFS filestore")
						return
					}

					key := datastore.NewKey(strings.TrimSuffix(ref.Path, "/"+dsfs.PackageFileDataset.String()))

					_, e := fs.Fetch(cafs.SourceAny, key)
					if e != nil {
						log.Debugf("error fetching file: %s", e.Error())
						return
					}

					e = fs.Pin(key, true)
					if e != nil {
						log.Debug(e.Error())
						return
					}
				}()
			}

		} else {
			log.Debug(err.Error())
		}
	}

	for _, remove := range changes.Deleted {
		if ref, err := repo.ParseDatasetRef(remove); err == nil {
			if err := n.Repo.RefCache().DeleteRef(ref); err != nil {
				log.Debug(err.Error())
			}
		} else {
			log.Debug(err.Error())
		}
	}

	return
}
