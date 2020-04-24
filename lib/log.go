package lib

import (
	"context"
	"fmt"
	"net/rpc"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
	reporef "github.com/qri-io/qri/repo/ref"
)

// LogRequests encapsulates business logic for the log
// of changes to datasets, think "git log"
// TODO (b5): switch to using an Instance instead of separate fields
type LogRequests struct {
	node *p2p.QriNode
	cli  *rpc.Client
}

// CoreRequestsName implements the Requets interface
func (r LogRequests) CoreRequestsName() string { return "log" }

// NewLogRequests creates a LogRequests pointer from either a repo
// or an rpc.Client
func NewLogRequests(node *p2p.QriNode, cli *rpc.Client) *LogRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both node and client supplied to NewLogRequests"))
	}
	return &LogRequests{
		node: node,
		cli:  cli,
	}
}

// LogParams defines parameters for the Log method
type LogParams struct {
	ListParams
	// Reference to data to fetch history for
	Ref string
}

// DatasetLogItem is a line item in a dataset logbook
type DatasetLogItem = logbook.DatasetLogItem

// Log returns the history of changes for a given dataset
func (r *LogRequests) Log(params *LogParams, res *[]DatasetLogItem) (err error) {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("LogRequests.Log", params, res))
	}
	ctx := context.TODO()

	if params.Ref == "" {
		return repo.ErrEmptyRef
	}
	ref, err := repo.ParseDatasetRef(params.Ref)
	if err != nil {
		return fmt.Errorf("'%s' is not a valid dataset reference", params.Ref)
	}
	// we only canonicalize the profile here, full dataset canonicalization
	// currently relies on repo's refstore, and the logbook may be a superset
	// of the refstore
	if err = repo.CanonicalizeProfile(r.node.Repo, &ref); err != nil {
		return err
	}

	// ensure valid limit value
	if params.Limit <= 0 {
		params.Limit = 25
	}
	// ensure valid offset value
	if params.Offset < 0 {
		params.Offset = 0
	}

	*res, err = base.DatasetLog(ctx, r.node.Repo, ref, params.Limit, params.Offset, true)
	return
}

// RefListParams encapsulates parameters for requests to a single reference
// that will produce a paginated result
type RefListParams struct {
	// String value of a reference
	Ref string
	// Pagination Parameters
	Offset, Limit int
}

// LogEntry is a record in a log of operations on a dataset
type LogEntry = logbook.LogEntry

// Logbook lists log entries for actions taken on a given dataset
func (r *LogRequests) Logbook(p *RefListParams, res *[]LogEntry) error {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("LogRequests.Logbook", p, res))
	}
	ctx := context.TODO()

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, &ref); err != nil {
		return err
	}

	book := r.node.Repo.Logbook()
	*res, err = book.LogEntries(ctx, reporef.ConvertToDsref(ref), p.Offset, p.Limit)
	return err
}

// PlainLogsParams enapsulates parameters for the PlainLogs methods
type PlainLogsParams struct {
	// no options yet
}

// PlainLogs is an alias for a human representation of a plain-old-data logbook
type PlainLogs = []logbook.PlainLog

// PlainLogs encodes the full logbook as human-oriented json
func (r *LogRequests) PlainLogs(p *PlainLogsParams, res *PlainLogs) (err error) {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("LogRequests.PlainLogs", p, res))
	}
	ctx := context.TODO()
	*res, err = r.node.Repo.Logbook().PlainLogs(ctx)
	return err
}

//////////////////////////////////////////////////////////////////////

// ReconstructRefsFromLogs
func (r *LogRequests) ReconstructRefsFromLogs(in *bool, out *bool) (err error) {
	if r.cli != nil {
		return checkRPCError(r.cli.Call("LogRequests.ReconstructRefsFromLogs", in, out))
	}
	ctx := context.TODO()
	plain, err := r.node.Repo.Logbook().PlainLogs(ctx)
	if err != nil {
		return err
	}

	refEntries := []reporef.DatasetRef{}

	for _, userLog := range plain {
		username, profileID := processUserLogOps(userLog.Ops)
		pid, err := profile.IDB58Decode(profileID)
		if err != nil {
			log.Errorf("decoding error: %s, input was %s", err, profileID)
			continue
		}

		refsLogs := userLog.Logs
		for _, dsLog := range refsLogs {

			datasetInfo := processDatasetLog(dsLog.Ops, dsLog.Logs)
			if datasetInfo == nil {
				// Dataset was deleted, don't add to refs
				continue
			}

			refEntries = append(refEntries, reporef.DatasetRef{
				Peername:  username,
				ProfileID: pid,
				Name:      datasetInfo.Name,
				Path:      datasetInfo.Path,
				Published: datasetInfo.Published,
			})
		}
	}

	err = r.node.Repo.ReplaceContent(refEntries)
	if err != nil {
		return err
	}

/*
func (rs *Refstore) save(refs repo.RefList) error {
	sort.Sort(refs)
	path := rs.basepath.filepath(rs.file)
	return ioutil.WriteFile(path, repo.FlatbufferBytes(refs), os.ModePerm)
}
*/

	return nil
}

func processUserLogOps(ops []logbook.PlainOp) (string, string) {
	username := ""
	profileID := ""
	for _, op := range ops {
		username = op.Name
		if len(op.AuthorID) == 46 {
			profileID = op.AuthorID
		}
	}
	return username, profileID
}

type datasetInfo struct {
	Name      string
	Path      string
	Published bool
}

func processDatasetLog(ops []logbook.PlainOp, logs []logbook.PlainLog) *datasetInfo {
	datasetType := ""
	datasetName := ""
	for k, op := range ops {
		if op.Type == "init" || op.Type == "remove" {
			datasetName = op.Name
			datasetType = op.Type
		} else {
			fmt.Printf("*** UNKNOWN DATASET OP %d: %s\n", k, op.Type)
		}
	}
	// Dataset deleted, return nil
	if datasetType == "remove" {
		return nil
	}

	if len(logs) != 1 {
		fmt.Printf("*** ONLY EXPECTED 1 LOG FOR SINGLE BRANCH\n")
	}
	branchLog := logs[0]
	pathRef := ""
	for _, op := range branchLog.Ops {
		pathRef = op.Ref
	}
	if len(branchLog.Logs) != 0 {
		fmt.Printf("*** EXPECTED BRANCH TO NOT HAVE LOGS\n")
	}
	return &datasetInfo{Name: datasetName, Path: pathRef, Published: false}
}
