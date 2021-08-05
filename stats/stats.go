// Package stats defines a stats provider service, wrapping a cache & stats
// component calculation
package stats

import (
	"context"
	"fmt"

	logger "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/detect"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/dsstats"
	"github.com/qri-io/qri/fsi"
)

var log = logger.Logger("stats")

// Service can generate an array of statistical info for a dataset
type Service struct {
	cache Cache
}

// New allocates a Stats service
func New(cache Cache) *Service {
	if cache == nil {
		cache = nilCache(false)
	}

	return &Service{
		cache: cache,
	}
}

// Stats gets the stats component for a dataset, possibly calculating
// by consuming the open dataset body file
func (s *Service) Stats(ctx context.Context, ds *dataset.Dataset) (*dataset.Stats, error) {
	if ds.Stats != nil {
		return ds.Stats, nil
	}

	key, err := s.cacheKey(ds)
	if err != nil {
		return nil, err
	}

	if sa, err := s.cache.GetStats(ctx, key); err == nil {
		log.Debugw("found cached stats", "key", key)
		return sa, nil
	}

	body := ds.BodyFile()
	if body == nil {
		return nil, fmt.Errorf("can't calculate stats. dataset has no body")
	}

	if ds.Structure == nil || ds.Structure.IsEmpty() {
		log.Debugw("inferring structure to calculate stats")
		ds.Structure = &dataset.Structure{}
		if err := detect.Structure(ds); err != nil {
			return nil, fmt.Errorf("inferring structure: %w", err)
		}
	}

	rdr, err := dsio.NewEntryReader(ds.Structure, ds.BodyFile())
	if err != nil {
		return nil, err
	}

	acc := dsstats.NewAccumulator(ds.Structure)
	err = dsio.EachEntry(rdr, func(i int, ent dsio.Entry, e error) error {
		return acc.WriteEntry(ent)
	})
	if err != nil {
		return nil, err
	}
	if err = acc.Close(); err != nil {
		return nil, err
	}

	sa := &dataset.Stats{
		Qri:   dataset.KindStats.String(),
		Stats: dsstats.ToMap(acc),
	}

	if cacheErr := s.cache.PutStats(ctx, key, sa); cacheErr != nil {
		log.Debugw("error caching stats", "path", ds.Path, "error", cacheErr)
	}

	return sa, nil
}

func (s *Service) cacheKey(ds *dataset.Dataset) (string, error) {
	if fsi.IsFSIPath(ds.Path) {
		// if the passed-in dataset is FSI-linked, use the body file
		// as a basis for the cache key
		// TODO(b5) - using only one file as a target means changing the structure
		// component can't invalidate the cache. We should be able to specify
		// an arbitrary number of target files for cache invalidation along with
		// a single canonical path
		bf := ds.BodyFile()
		if bf == nil {
			return "", fmt.Errorf("A Body File is required to calculate stats")
		}
		log.Debugw("dataset is FSI-linked, using body key", "path", ds.Path, "bodyPath", bf.FullPath())
		return bf.FullPath(), nil
	}

	return ds.Path, nil
}
