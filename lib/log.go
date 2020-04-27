package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/qri/base"
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
	Ref string
}

// DatasetLogItem is a line item in a dataset logbook
type DatasetLogItem = logbook.DatasetLogItem

// Log returns the history of changes for a given dataset
func (m *LogMethods) Log(params *LogParams, res *[]DatasetLogItem) error {
	if m.inst.rpc != nil {
		return checkRPCError(m.inst.rpc.Call("LogMethods.Log", params, res))
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
	if err = repo.CanonicalizeProfile(m.inst.node.Repo, &ref); err != nil {
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

	*res, err = base.DatasetLog(ctx, m.inst.node.Repo, ref, params.Limit, params.Offset, true)
	return err
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
