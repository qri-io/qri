package lib

import (
	"context"

	"github.com/qri-io/qri/changes"
	"github.com/qri-io/qri/dsref"
)

// ChangeReportParams defines parameters for diffing two sources
type ChangeReportParams struct {
	LeftRefstr  string `schema:"leftRef" json:"leftRef"`
	RightRefstr string `schema:"rightRef" json:"rightRef"`
}

// ChangeReport is a simple utility type declaration
type ChangeReport = changes.ChangeReportResponse

// ChangeReport resolves the requested datasets and tries to generate a change report
func (m *DatasetMethods) ChangeReport(ctx context.Context, p *ChangeReportParams) (*ChangeReport, error) {
	got, _, err := m.inst.Dispatch(ctx, dispatchMethodName(m, "changereport"), p)
	if res, ok := got.(*ChangeReport); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// ChangeReport generates report of changes between two datasets
func (datasetImpl) ChangeReport(scope scope, p *ChangeReportParams) (*ChangeReport, error) {
	ctx := scope.Context()
	reportSource := ""

	right, _, err := scope.ParseAndResolveRef(ctx, p.RightRefstr, reportSource)
	if err != nil {
		return nil, err
	}

	var left dsref.Ref
	if p.LeftRefstr != "" {
		if left, _, err = scope.ParseAndResolveRef(ctx, p.LeftRefstr, reportSource); err != nil {
			return nil, err
		}
	} else {
		left = dsref.Ref{Username: right.Username, Name: right.Name}
	}

	return changes.New(scope.inst, scope.inst.stats).Report(ctx, left, right, reportSource)
}
