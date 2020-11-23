package changes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	golog "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/tabular"
	"github.com/qri-io/qri/dsref"
	qerr "github.com/qri-io/qri/errors"
	"github.com/qri-io/qri/fsi"
	"github.com/qri-io/qri/stats"
)

var (
	log = golog.Logger("changes")
)

// ChangeReportComponent is a generic component used to populate the change report
type ChangeReportComponent struct {
	Left  interface{}            `json:"left"`
	Right interface{}            `json:"right"`
	About map[string]interface{} `json:"about,omitempty"`
}

// ChangeReportDeltaComponent is a subcomponent that can hold
// delta information between left and right
type ChangeReportDeltaComponent struct {
	ChangeReportComponent
	Title string      `json:"title,omitempty"`
	Delta interface{} `json:"delta"`
}

// StatsChangeComponent represents the stats change report
type StatsChangeComponent struct {
	Summary *ChangeReportDeltaComponent   `json:"summary"`
	Columns []*ChangeReportDeltaComponent `json:"columns"`
}

// ChangeReportResponse is the result of a call to changereport
type ChangeReportResponse struct {
	VersionInfo *ChangeReportComponent `json:"version_info,omitempty"`
	Commit      *ChangeReportComponent `json:"commit,omitempty"`
	Meta        *ChangeReportComponent `json:"meta,omitempty"`
	Readme      *ChangeReportComponent `json:"readme,omitempty"`
	Structure   *ChangeReportComponent `json:"structure,omitempty"`
	Transform   *ChangeReportComponent `json:"transform,omitempty"`
	Stats       *StatsChangeComponent  `json:"stats,omitempty"`
}

// StatsChangeSummaryFields represents the stats summary
type StatsChangeSummaryFields struct {
	Entries int `json:"entries"`
	Columns int `json:"columns"`
	// NullValues int `json:"nullValues"`
	TotalSize int `json:"totalSize"`
}

// EmptyObject is used mostly as a placeholder in palces where it is required
// that a key is present in the response even if empty and not be nil
type EmptyObject map[string]interface{}

// Service can generate a change report between two datasets
type Service struct {
	loader dsref.Loader
	stats  *stats.Service
}

// New allocates a Change service
func New(loader dsref.Loader, stats *stats.Service) *Service {
	return &Service{
		loader: loader,
		stats:  stats,
	}
}

func (svc *Service) parseColumns(colItems *tabular.Columns, data *dataset.Dataset) (interface{}, error) {
	var sErr error
	if data.Structure != nil {
		*colItems, _, sErr = tabular.ColumnsFromJSONSchema(data.Structure.Schema)
		if sErr != nil {
			return nil, sErr
		}
		return StatsChangeSummaryFields{
			Entries:   data.Structure.Entries,
			Columns:   len(*colItems),
			TotalSize: data.Structure.Length,
		}, nil
	}
	return EmptyObject{}, nil
}

// maybeLoadStats attempts to load stats if not alredy present
// errors out if it fails as stats are required and some datasets might not yet have
// a stats component attached to it
func (svc *Service) maybeLoadStats(ctx context.Context, ds *dataset.Dataset) error {
	if ds.Stats != nil {
		return nil
	}
	var statsErr error
	ds.Stats, statsErr = svc.stats.Stats(ctx, ds)
	if statsErr != nil {
		return qerr.New(statsErr, "missing stats components")
	}
	return nil
}

// parseStats uses json serializing > deserializing to easily parse the stats
// interface as we have little type safety in the dataset.stats component right now
func (svc *Service) parseStats(ds *dataset.Dataset) ([]EmptyObject, error) {
	statsStr, err := json.Marshal(ds.Stats.Stats)
	if err != nil {
		log.Debugf("failed to load stats: %s", err.Error())
		return nil, qerr.New(err, "failed to load stats")
	}
	stats := []EmptyObject{}
	err = json.Unmarshal(statsStr, &stats)
	if err != nil {
		log.Debugf("failed to parse stats: %s", err.Error())
		return nil, qerr.New(err, "failed to parse stats")
	}

	return stats, nil
}

func (svc *Service) statsDiff(ctx context.Context, leftDs *dataset.Dataset, rightDs *dataset.Dataset) (*StatsChangeComponent, error) {
	res := &StatsChangeComponent{}

	res.Summary = &ChangeReportDeltaComponent{
		ChangeReportComponent: ChangeReportComponent{},
	}

	var leftColItems, rightColItems tabular.Columns
	var sErr error
	res.Summary.Left, sErr = svc.parseColumns(&leftColItems, leftDs)
	if sErr != nil {
		return &StatsChangeComponent{}, sErr
	}
	leftColCount := len(leftColItems)

	res.Summary.Right, sErr = svc.parseColumns(&rightColItems, rightDs)
	if sErr != nil {
		return &StatsChangeComponent{}, sErr
	}
	rightColCount := len(rightColItems)

	if leftDs.Structure != nil && rightDs.Structure != nil {
		res.Summary.Delta = StatsChangeSummaryFields{
			Entries:   res.Summary.Right.(StatsChangeSummaryFields).Entries - res.Summary.Left.(StatsChangeSummaryFields).Entries,
			Columns:   rightColCount - leftColCount,
			TotalSize: res.Summary.Right.(StatsChangeSummaryFields).TotalSize - res.Summary.Left.(StatsChangeSummaryFields).TotalSize,
		}
	} else if leftDs.Structure != nil {
		res.Summary.Delta = StatsChangeSummaryFields{
			Entries:   -res.Summary.Left.(StatsChangeSummaryFields).Entries,
			Columns:   rightColCount - leftColCount,
			TotalSize: -res.Summary.Left.(StatsChangeSummaryFields).TotalSize,
		}
	} else if rightDs.Structure != nil {
		res.Summary.Delta = StatsChangeSummaryFields{
			Entries:   res.Summary.Right.(StatsChangeSummaryFields).Entries,
			Columns:   rightColCount - leftColCount,
			TotalSize: res.Summary.Right.(StatsChangeSummaryFields).TotalSize,
		}
	} else {
		res.Summary.Delta = StatsChangeSummaryFields{
			Entries:   0,
			Columns:   0,
			TotalSize: 0,
		}
	}

	err := svc.maybeLoadStats(ctx, leftDs)
	if err != nil {
		return nil, err
	}
	err = svc.maybeLoadStats(ctx, rightDs)
	if err != nil {
		return nil, err
	}

	leftStats, err := svc.parseStats(leftDs)
	if err != nil {
		return nil, err
	}
	rightStats, err := svc.parseStats(rightDs)
	if err != nil {
		return nil, err
	}

	res.Columns, err = svc.matchColumns(leftColCount, rightColCount, leftColItems, rightColItems, leftStats, rightStats)
	if err != nil {
		log.Debugf("failed to calculate stats change report: %s", err.Error())
		return nil, qerr.New(err, "failed to calculate stats change report")
	}

	return res, nil
}

// matchColumns attempts to match up columns from the left and right side based on the column name
// this is not ideal as datasets without a header have generic column names and in case of adding a column
// before the end might shift the alignment and break comparison due to type differences of columns which
// are not properly handled yet
func (svc *Service) matchColumns(leftColCount, rightColCount int, leftColItems, rightColItems tabular.Columns, leftStats, rightStats []EmptyObject) ([]*ChangeReportDeltaComponent, error) {
	maxColCount := leftColCount
	if rightColCount > maxColCount {
		maxColCount = rightColCount
	}

	columns := make([]*ChangeReportDeltaComponent, maxColCount)

	type cIndex struct {
		LPos int
		RPos int
	}

	colIndex := map[string]*cIndex{}
	for i := 0; i < maxColCount; i++ {
		if i < leftColCount {
			if c, ok := colIndex[leftColItems[i].Title]; ok && c != nil {
				colIndex[leftColItems[i].Title].LPos = i
			} else {
				colIndex[leftColItems[i].Title] = &cIndex{
					LPos: i,
					RPos: -1,
				}
			}
		}
		if i < rightColCount {
			if c, ok := colIndex[rightColItems[i].Title]; ok && c != nil {
				colIndex[rightColItems[i].Title].RPos = i
			} else {
				colIndex[rightColItems[i].Title] = &cIndex{
					LPos: -1,
					RPos: i,
				}
			}
		}
	}

	i := 0
	for k, v := range colIndex {
		columns[i] = &ChangeReportDeltaComponent{
			Title: k,
		}
		var lCol, rCol *tabular.Column
		if v.LPos >= 0 {
			columns[i].Left = leftStats[v.LPos]
			lCol = &leftColItems[v.LPos]
		} else {
			columns[i].Left = EmptyObject{}
		}
		if v.RPos >= 0 {
			columns[i].Right = rightStats[v.RPos]
			rCol = &rightColItems[v.RPos]
		} else {
			columns[i].Right = EmptyObject{}
		}
		deltaCol, aboutCol, err := svc.columnStatsDelta(columns[i].Left, columns[i].Right, lCol, rCol, v.LPos >= 0, v.RPos >= 0)
		if err != nil {
			log.Debugf("error calculating stats delta: %s", err.Error())
			return nil, qerr.New(err, fmt.Sprintf("failed to calculate stats column delta for %q", columns[i].Title))
		}
		columns[i].Delta = deltaCol
		columns[i].About = aboutCol
		i++
	}

	return columns, nil
}

func parseStatsMap(stats interface{}) (map[string]interface{}, error) {
	statsMap := map[string]interface{}{}
	serialized, err := json.Marshal(stats)
	if err != nil {
		log.Debugf("error serializing stats")
		return nil, err
	}
	err = json.Unmarshal(serialized, &statsMap)
	if err != nil {
		log.Debugf("error deserializing stats")
		return nil, err
	}
	return statsMap, nil
}

func (svc *Service) columnStatsDelta(left, right interface{}, lCol, rCol *tabular.Column, hasLeft, hasRight bool) (map[string]interface{}, map[string]interface{}, error) {
	var deltaCol map[string]interface{}
	aboutCol := map[string]interface{}{}
	var leftStatsMap, rightStatsMap map[string]interface{}
	var err error
	if hasLeft {
		leftStatsMap, err = parseStatsMap(left)
		if err != nil {
			log.Debugf("error parsing stats map")
			return nil, nil, err
		}
	}
	if hasRight {
		rightStatsMap, err = parseStatsMap(right)
		if err != nil {
			log.Debugf("error parsing stats map")
			return nil, nil, err
		}
	}

	//determine shape
	if (!hasRight || (hasRight && (rCol.Type.HasType("number") || rCol.Type.HasType("integer")))) &&
		(!hasLeft || (hasLeft && (lCol.Type.HasType("number") || lCol.Type.HasType("integer")))) {
		deltaCol = map[string]interface{}{
			"count":  float64(0),
			"max":    float64(0),
			"min":    float64(0),
			"median": float64(0),
			"mean":   float64(0),
		}
	} else if (!hasRight || (hasRight && rCol.Type.HasType("string"))) && (!hasLeft || (hasLeft && lCol.Type.HasType("string"))) {
		deltaCol = map[string]interface{}{
			"count":     float64(0),
			"maxLength": float64(0),
			"minLength": float64(0),
			"unique":    float64(0),
		}
	} else if (!hasRight || (hasRight && rCol.Type.HasType("boolean"))) && (!hasLeft || (hasLeft && lCol.Type.HasType("boolean"))) {
		deltaCol = map[string]interface{}{
			"count":      float64(0),
			"trueCount":  float64(0),
			"falseCount": float64(0),
		}
	} else {
		log.Debugf("incompatible column types: %+v / %+v", rCol.Type, lCol.Type)
		// TODO(arqu): improve handling of columns with different types
		return nil, nil, errors.New("incompatible column types")
	}

	// assign values
	for k := range deltaCol {
		if hasLeft {
			if leftStatsMap[k] == nil {
				log.Debugf("%s was nil", k)
			} else {
				deltaCol[k] = deltaCol[k].(float64) - leftStatsMap[k].(float64)
			}
		}
		if hasRight {
			if rightStatsMap[k] == nil {
				log.Debugf("%s was nil", k)
			} else {
				deltaCol[k] = deltaCol[k].(float64) + rightStatsMap[k].(float64)
			}
		}
	}

	if hasLeft && !hasRight {
		aboutCol["status"] = fsi.STRemoved
	} else if !hasLeft && hasRight {
		aboutCol["status"] = fsi.STAdd
	} else if hasLeft && hasRight {
		sum := float64(0)
		for k := range deltaCol {
			sum += deltaCol[k].(float64)
		}
		if sum == 0 {
			aboutCol["status"] = fsi.STUnmodified
		} else {
			aboutCol["status"] = fsi.STChange
		}
	} else {
		aboutCol["status"] = fsi.STMissing
	}

	return deltaCol, aboutCol, nil
}

// Report computes the change report of two sources
// This takes some assumptions - we work only with tabular data, with header rows and functional structure.json
func (svc *Service) Report(ctx context.Context, leftRef, rightRef dsref.Ref, loadSource string) (*ChangeReportResponse, error) {
	leftDs, err := svc.loader.LoadDataset(ctx, leftRef, loadSource)
	if err != nil {
		return nil, err
	}
	if rightRef.Path == "" {
		rightRef.Path = leftDs.PreviousPath
	}
	rightDs, err := svc.loader.LoadDataset(ctx, rightRef, loadSource)
	if err != nil {
		return nil, err
	}

	res := &ChangeReportResponse{}

	leftVi := dsref.ConvertDatasetToVersionInfo(leftDs)
	rightVi := dsref.ConvertDatasetToVersionInfo(rightDs)

	res.VersionInfo = &ChangeReportComponent{}
	res.VersionInfo.Left = leftVi
	res.VersionInfo.Right = rightVi
	res.VersionInfo.About = EmptyObject{}

	if leftVi.Path == rightVi.Path {
		res.VersionInfo.About["status"] = fsi.STUnmodified
	} else {
		res.VersionInfo.About["status"] = fsi.STChange
	}

	if leftDs.Commit != nil || rightDs.Commit != nil {
		res.Commit = &ChangeReportComponent{}
		if leftDs.Commit != nil {
			res.Commit.Left = leftDs.Commit
		} else {
			res.Commit.Left = EmptyObject{}
		}
		if rightDs.Commit != nil {
			res.Commit.Right = rightDs.Commit
		} else {
			res.Commit.Right = EmptyObject{}
		}
		res.Commit.About = EmptyObject{}

		if leftDs.Commit != nil && rightDs.Commit == nil {
			res.Commit.About["status"] = fsi.STRemoved
		} else if leftDs.Commit == nil && rightDs.Commit != nil {
			res.Commit.About["status"] = fsi.STAdd
		} else if leftDs.Commit != nil && rightDs.Commit != nil {
			if leftDs.Commit.Path == rightDs.Commit.Path {
				res.Commit.About["status"] = fsi.STUnmodified
			} else {
				res.Commit.About["status"] = fsi.STChange
			}
		} else {
			res.Commit.About["status"] = fsi.STMissing
		}
	}

	if leftDs.Meta != nil || rightDs.Meta != nil {
		res.Meta = &ChangeReportComponent{}
		hasLeftMeta := leftDs.Meta != nil && !leftDs.Meta.IsEmpty()
		hasRightMeta := rightDs.Meta != nil && !rightDs.Meta.IsEmpty()

		if hasLeftMeta {
			res.Meta.Left = leftDs.Meta
		} else {
			res.Meta.Left = EmptyObject{}
		}
		if hasRightMeta {
			res.Meta.Right = rightDs.Meta
		} else {
			res.Meta.Right = EmptyObject{}
		}
		res.Meta.About = EmptyObject{}

		if hasLeftMeta && !hasRightMeta {
			res.Meta.About["status"] = fsi.STRemoved
		} else if !hasLeftMeta && hasRightMeta {
			res.Meta.About["status"] = fsi.STAdd
		} else if hasLeftMeta && hasRightMeta {
			if leftDs.Meta.Path == rightDs.Meta.Path {
				res.Meta.About["status"] = fsi.STUnmodified
			} else {
				res.Meta.About["status"] = fsi.STChange
			}
		} else {
			res.Meta.About["status"] = fsi.STMissing
		}
	}

	if leftDs.Readme != nil || rightDs.Readme != nil {
		res.Readme = &ChangeReportComponent{}
		if leftDs.Readme != nil {
			res.Readme.Left = string(leftDs.Readme.ScriptBytes)
		} else {
			res.Readme.Left = ""
		}
		if rightDs.Readme != nil {
			res.Readme.Right = string(rightDs.Readme.ScriptBytes)
		} else {
			res.Readme.Right = ""
		}
		res.Readme.About = EmptyObject{}

		if res.Readme.Left != "" && res.Readme.Right == "" {
			res.Readme.About["status"] = fsi.STRemoved
		} else if res.Readme.Left == "" && res.Readme.Right != "" {
			res.Readme.About["status"] = fsi.STAdd
		} else if res.Readme.Left != "" && res.Readme.Right != "" {
			if res.Readme.Left == res.Readme.Right {
				res.Readme.About["status"] = fsi.STUnmodified
			} else {
				res.Readme.About["status"] = fsi.STChange
			}
		} else {
			res.Readme.About["status"] = fsi.STMissing
		}
	}

	if leftDs.Structure != nil || rightDs.Structure != nil {
		res.Structure = &ChangeReportComponent{}
		if leftDs.Structure != nil {
			if leftDs.Structure.Format != "csv" {
				return nil, errors.New("changes are only supported for CSV datasets")
			}
			res.Structure.Left = leftDs.Structure
		} else {
			res.Structure.Left = EmptyObject{}
		}
		if rightDs.Structure != nil {
			if rightDs.Structure.Format != "csv" {
				return nil, errors.New("changes are only supported for CSV datasets")
			}
			res.Structure.Right = rightDs.Structure
		} else {
			res.Structure.Right = EmptyObject{}
		}
		res.Structure.About = EmptyObject{}

		if leftDs.Structure != nil && rightDs.Structure == nil {
			res.Structure.About["status"] = fsi.STRemoved
		} else if leftDs.Structure == nil && rightDs.Structure != nil {
			res.Structure.About["status"] = fsi.STAdd
		} else if leftDs.Structure != nil && rightDs.Structure != nil {
			if leftDs.Structure.Path == rightDs.Structure.Path {
				res.Structure.About["status"] = fsi.STUnmodified
			} else {
				res.Structure.About["status"] = fsi.STChange
			}
		} else {
			res.Structure.About["status"] = fsi.STMissing
		}
	}

	if leftDs.Transform != nil || rightDs.Transform != nil {
		res.Transform = &ChangeReportComponent{}
		if leftDs.Transform != nil {
			res.Transform.Left = string(leftDs.Transform.ScriptBytes)
		} else {
			res.Transform.Left = ""
		}
		if rightDs.Transform != nil {
			res.Transform.Right = string(rightDs.Transform.ScriptBytes)
		} else {
			res.Transform.Right = ""
		}
		res.Transform.About = EmptyObject{}

		if res.Transform.Left != "" && res.Transform.Right == "" {
			res.Transform.About["status"] = fsi.STRemoved
		} else if res.Transform.Left == "" && res.Transform.Right != "" {
			res.Transform.About["status"] = fsi.STAdd
		} else if res.Transform.Left != "" && res.Transform.Right != "" {
			if res.Transform.Left == res.Transform.Right {
				res.Transform.About["status"] = fsi.STUnmodified
			} else {
				res.Transform.About["status"] = fsi.STChange
			}
		} else {
			res.Transform.About["status"] = fsi.STMissing
		}
	}

	res.Stats, err = svc.statsDiff(ctx, leftDs, rightDs)
	if err != nil {
		return nil, err
	}
	return res, nil
}
