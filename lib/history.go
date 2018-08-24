package lib

import (
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/actions"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// HistoryRequests encapsulates business logic for the log
// of changes to datasets, think "git log"
type HistoryRequests struct {
	repo actions.Dataset
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
		repo: actions.Dataset{r},
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
	if err = DefaultSelectedRef(d.repo.Repo, &ref); err != nil {
		return
	}

	// TODO: Cleanup how this treats non-local datasetrefs. It is doing a bunch of work below
	// to request remote data, but it's weird how it does so - once CanonicalizeDatasetRef returns
	// ErrNotFound, we know that the data is remote. Don't need this callback function, just
	// call RequestDatasetLog. Also, a lot of other functions need something like this; this
	// pattern should be generalized.
	err = repo.CanonicalizeDatasetRef(d.repo, &ref)
	if err != nil && err != repo.ErrNotFound {
		log.Debug(err.Error())
		return err
	}

	getRemote := func(err error) error {
		if d.Node != nil {
			rlog, err := d.Node.RequestDatasetLog(ref)
			if err != nil {
				log.Debug(err.Error())
				return err
			}

			*res = *rlog
			return nil
		}
		return err
	}

	_, err = d.repo.GetRef(ref)
	if err != nil {
		err = fmt.Errorf("error getting reference '%s': %s", ref, err.Error())
		return getRemote(err)
	}

	rlog := []repo.DatasetRef{}
	limit := params.Limit

	for {
		if err = d.repo.ReadDataset(&ref); err != nil {
			log.Debug(err.Error())
			return fmt.Errorf("error adding datasets to log: %s", err.Error())
		}
		rlog = append(rlog, ref)

		limit--
		if limit == 0 || ref.Dataset.PreviousPath == "" {
			break
		}
		ref.Path = ref.Dataset.PreviousPath
	}

	*res = rlog
	return nil
}
