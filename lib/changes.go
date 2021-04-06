package lib

import (
	"context"

	"github.com/qri-io/qri/changes"
	"github.com/qri-io/qri/dsref"
)

// ChangeReportParams defines parameters for diffing two sources
type ChangeReportParams struct {
	LeftRef  string `schema:"leftRef" json:"leftRef"`
	RightRef string `schema:"rightRef" json:"rightRef"`
}

// ChangeReport is a simple utility type declaration
type ChangeReport = changes.ChangeReportResponse

// ChangeReport resolves the requested datasets and tries to generate a change report
func (m DatasetMethods) ChangeReport(ctx context.Context, p *ChangeReportParams) (*ChangeReport, error) {
	got, _, err := m.d.Dispatch(ctx, dispatchMethodName(m, "changereport"), p)
	if res, ok := got.(*ChangeReport); ok {
		return res, err
	}
	return nil, dispatchReturnError(got, err)
}

// ChangeReport generates report of changes between two datasets
func (datasetImpl) ChangeReport(scope scope, p *ChangeReportParams) (*ChangeReport, error) {
	ctx := scope.Context()

	right, location, err := scope.ParseAndResolveRef(ctx, p.RightRef)
	if err != nil {
		return nil, err
	}

	var left dsref.Ref
	if p.LeftRef != "" {
		if left, _, err = scope.ParseAndResolveRef(ctx, p.LeftRef); err != nil {
			return nil, err
		}
	} else {
		left = dsref.Ref{Username: right.Username, Name: right.Name}
	}

	return changes.New(scope.Loader(), scope.Stats()).Report(ctx, left, right, location)
}
