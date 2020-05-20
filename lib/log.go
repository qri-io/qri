package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/logbook"
	"github.com/qri-io/qri/repo"
	reporef "github.com/qri-io/qri/repo/ref"
)

// LogMethods extends a lib.Instance with business logic for working with lists
// of dataset versions. think "git log".
type LogMethods struct {
	inst *Instance
}

// CoreRequestsName implements the Requets interface
func (m LogMethods) CoreRequestsName() string { return "log" }

// NewLogMethods creates a LogMethods pointer from either a repo
// or an rpc.Client
func NewLogMethods(inst *Instance) *LogMethods {
	return &LogMethods{
		inst: inst,
	}
}

// LogParams defines parameters for the Log method
type LogParams struct {
	ListParams
	// Reference to data to fetch history for
	Ref    string
	Pull   bool
	Source string
}

// DatasetLogItem is a line item in a dataset logbook
type DatasetLogItem = logbook.DatasetLogItem

// Log returns the history of changes for a given dataset
func (m *LogMethods) Log(params *LogParams, res *[]DatasetLogItem) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("LogMethods.Log", params, res))
	}
	ctx := context.TODO()

	// ensure valid limit value
	if params.Limit <= 0 {
		params.Limit = 25
	}
	// ensure valid offset value
	if params.Offset < 0 {
		params.Offset = 0
	}

	if params.Pull {
		switch params.Source {
		case "":
			params.Source = "network"
		case "local":
			return fmt.Errorf("cannot pull with only local source")
		}
	}

	ref, source, err := m.inst.ParseAndResolveRef(ctx, params.Ref, params.Source)
	if err != nil {
		return err
	}

	if source == "" {
		// local resolution
		*res, err = base.DatasetLog(ctx, m.inst.repo, ref, params.Limit, params.Offset, true)
		return err
	}

	logs, err := m.inst.remoteClient.FetchLogs(ctx, ref, source)
	if err != nil {
		return err
	}

	// TODO (b5) - FetchLogs currently returns oplogs arranged in user > dataset > branch
	// hierarchy, and we need to descend to the branch oplog to get commit history
	// info. It might be nicer if FetchLogs instead returned the branch oplog, but
	// with .Parent() fields loaded & connected
	if len(logs.Logs) > 0 {
		logs = logs.Logs[0]
		if len(logs.Logs) > 0 {
			logs = logs.Logs[0]
		}
	}

	items := logbook.ConvertLogsToItems(logs, ref)
	log.Debugf("found %d items: %v", len(items), items)
	if len(items) == 0 {
		return repo.ErrNoHistory
	}

	for i, item := range items {
		local, hasErr := m.inst.Repo().Store().Has(ctx, item.Path)
		if hasErr != nil {
			continue
		}
		items[i].Foreign = !local

		if local {
			if ds, err := dsfs.LoadDataset(ctx, m.inst.repo.Store(), item.Path); err == nil {
				if ds.Commit != nil {
					items[i].CommitMessage = ds.Commit.Message
				}
			}
		}
	}

	// TODO (b5) - store logs on pull
	// if params.Pull {
	// }

	*res = items
	return nil
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
func (m *LogMethods) Logbook(p *RefListParams, res *[]LogEntry) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("LogMethods.Logbook", p, res))
	}
	ctx := context.TODO()

	ref, err := repo.ParseDatasetRef(p.Ref)
	if err != nil {
		return err
	}
	if err = repo.CanonicalizeDatasetRef(m.inst.node.Repo, &ref); err != nil {
		return err
	}

	book := m.inst.node.Repo.Logbook()
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
func (m *LogMethods) PlainLogs(p *PlainLogsParams, res *PlainLogs) error {
	var err error
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("LogMethods.PlainLogs", p, res))
	}
	ctx := context.TODO()
	*res, err = m.inst.repo.Logbook().PlainLogs(ctx)
	return err
}
