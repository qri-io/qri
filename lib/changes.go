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
	reportSource := ""

	if m.inst.http != nil {
		res := &ChangeReport{}
		err := m.inst.http.Call(ctx, AEChanges, p, res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	right, _, err := m.inst.ParseAndResolveRef(ctx, p.RightRefstr, reportSource)
	if err != nil {
		return nil, err
	}

	var left dsref.Ref
	if p.LeftRefstr != "" {
		if left, _, err = m.inst.ParseAndResolveRef(ctx, p.LeftRefstr, reportSource); err != nil {
			return nil, err
		}
	} else {
		left = dsref.Ref{Username: right.Username, Name: right.Name}
	}

	return changes.New(m.inst, m.inst.stats).Report(ctx, left, right, reportSource)
}
