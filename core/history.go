package core

import (
	"fmt"
	"net/rpc"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// HistoryRequests encapsulates business logic for the log
// of changes to datasets, think "git log"
type HistoryRequests struct {
	repo repo.Repo
	cli  *rpc.Client
	Node *p2p.QriNode
}

// CoreRequestsName implements the Requets interface
func (d HistoryRequests) CoreRequestsName() string { return "history" }

// NewHistoryRequests creates a HistoryRequests pointer from either a repo
// or an rpc.Client
func NewHistoryRequests(r repo.Repo, cli *rpc.Client) *HistoryRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewHistoryRequests"))
	}
	return &HistoryRequests{
		repo: r,
		cli:  cli,
	}
}

// LogParams defines parameters for the Log method
type LogParams struct {
	ListParams
	// Reference to data to fetch history for
	Ref repo.DatasetRef
}

// Log returns the history of changes for a given dataset
func (d *HistoryRequests) Log(params *LogParams, res *[]repo.DatasetRef) (err error) {
	if d.cli != nil {
		return d.cli.Call("HistoryRequests.Log", params, res)
	}

	ref := params.Ref
	err = repo.CanonicalizeDatasetRef(d.repo, &ref)
	if err != nil {
		return err
	}
	if ref.Path == "" && (ref.Name == "" && ref.Peername == "") {
		return fmt.Errorf("either path or peername/name is required")
	}

	// ref := &repo.DatasetRef{Peername: params.ListParams.Peername, Name: params.Name, Path: params.Path.String()}

	getRemote := func(err error) error {
		if d.Node != nil {
			log, err := d.Node.RequestDatasetLog(ref)
			if err != nil {
				return err
			}

			*res = *log
			return nil
		}
		return err
	}

	got, err := d.repo.GetRef(ref)
	if err != nil {
		err = fmt.Errorf("error getting reference '%s': %s", ref, err.Error())
		return getRemote(err)
	}
	ref = got

	log := []repo.DatasetRef{}
	limit := params.Limit

	for {
		ref.Dataset, err = d.repo.GetDataset(datastore.NewKey(ref.Path))
		if err != nil {
			return fmt.Errorf("error adding datasets to log: %s", err.Error())
		}
		log = append(log, ref)

		limit--
		if limit == 0 || ref.Dataset.PreviousPath == "" {
			break
		}
		ref.Path = ref.Dataset.PreviousPath
	}

	*res = log
	return nil
}
