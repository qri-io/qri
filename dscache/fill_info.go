package dscache

import (
	"context"
	"fmt"
	"strings"

	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/base/fill"
)

// fillInfoForDatasets iterates over the dsInfo list, looks up each dataset and adds relevent info
// from dsfs. If there are errors loading any datasets, we keep going, collecting such errors
// until the list iteration is done. Returns nil if there are no errors.
func fillInfoForDatasets(ctx context.Context, store cafs.Filestore, filesys qfs.Filesystem, dsInfoList []*dsInfo) error {
	collector := fill.NewErrorCollector()
	for _, info := range dsInfoList {
		if info.HeadRef == "" {
			continue
		}
		ds, err := dsfs.LoadDataset(ctx, store, info.HeadRef)
		if err != nil {
			collector.Add(fmt.Errorf("for initID %q: %s", info.InitID, err))
			continue
		}
		if ds.Meta != nil {
			info.MetaTitle = ds.Meta.Title
			info.ThemeList = strings.Join(ds.Meta.Theme, ",")
		}
		if ds.Structure != nil {
			info.BodyRows = ds.Structure.Entries
			info.BodySize = int64(ds.Structure.Length)
			info.BodyFormat = ds.Structure.Format
			info.NumErrors = ds.Structure.ErrCount
		}
		if ds.Commit != nil {
			info.CommitTime = ds.Commit.Timestamp
			info.CommitTitle = ds.Commit.Title
			info.CommitMessage = ds.Commit.Message
		}
	}
	return collector.AsSingleError()
}
