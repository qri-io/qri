package p2p

// func (n *QriNode) syncProfile(pid peer.ID) error {
// 	changes := Events{}
// 	if err := json.Unmarshal(msg.Body, &changes); err != nil {
// 	 log.Debug(err.Error())
// 	 return
// 	}

// 	for _, add := range changes.Created {
// 	 if ref, err := repo.ParseDatasetRef(add); err == nil {
// 	   if err := n.Repo.RefCache().PutRef(ref); err != nil {
// 	     log.Debug(err.Error())
// 	   }

// 	   if ref.PeerID == pro.ID && n.selfReplication == "full" {
// 	     log.Infof("%s %s self replicating dataset %s from %s", n.ID.Pretty(), pro.ID, ref.String(), msg.Initiator.Pretty())

// 	     if err = n.Repo.PutRef(ref); err != nil {
// 	       log.Debug(err.Error())
// 	       return
// 	     }

// 	     go func() {
// 	       // TODO - this is pulled out of core.DatasetRequests.Add
// 	       // we should find a common place for this code & have both add & this func call it
// 	       // possibly in / subpackage of repo?
// 	       fs, ok := n.Repo.Store().(*ipfs.Filestore)
// 	       if !ok {
// 	         log.Debug("can only add datasets when running an IPFS filestore")
// 	         return
// 	       }

// 	       key := datastore.NewKey(strings.TrimSuffix(ref.Path, "/"+dsfs.PackageFileDataset.String()))

// 	       _, e := fs.Fetch(cafs.SourceAny, key)
// 	       if e != nil {
// 	         log.Debugf("error fetching file: %s", e.Error())
// 	         return
// 	       }

// 	       e = fs.Pin(key, true)
// 	       if e != nil {
// 	         log.Debug(e.Error())
// 	         return
// 	       }
// 	     }()
// 	   }

// 	 } else {
// 	   log.Debug(err.Error())
// 	 }
// 	}

// 	for _, remove := range changes.Deleted {
// 	 if ref, err := repo.ParseDatasetRef(remove); err == nil {
// 	   if err := n.Repo.RefCache().DeleteRef(ref); err != nil {
// 	     log.Debug(err.Error())
// 	   }
// 	 } else {
// 	   log.Debug(err.Error())
// 	 }
// 	}
// }
