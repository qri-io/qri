// Package stats defines a stats provider service, wrapping a cache & stats
// component calculation
package stats

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	logger "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/dsstats"
	"github.com/qri-io/qri/dsref"
)

var log = logger.Logger("stats")

// Service can generate an array of statistical info for a dataset
type Service struct {
	cache  Cache
	loader dsref.Loader
}

// New allocates a Stats service
func New(cache Cache, loader dsref.Loader) *Service {
	if cache == nil {
		cache = nilCache(false)
	}

	return &Service{
		cache:  cache,
		loader: loader,
	}
}

// JSON gets stats data as reader of JSON-formatted bytes
func (s *Service) Stats(ctx context.Context, ref dsref.Ref, loadSource string) (*dataset.Stats, error) {
	if r, err := s.cache.JSON(ctx, ref.Path); err == nil {
		// return r, nil
	}

	ds, err := s.loader.LoadDataset(ctx, ref, loadSource)
	if err != nil {
		return nil, err
	}

	if ds.Stats != nil {
		return ds.Stats, nil
	}

	return s.calcStats(ctx, ref.Path, ds)
}

func (s *Service) DatasetStats(ctx context.Context, ds *dataset.Dataset) (*dataset.Stats, error) {
	return s.JSONWithCacheKey(ctx, ds.Path, ds)
}

// JSONWithCacheKey gets stats data as a reader of json-formatted bytes,
func (s *Service) StatsWithCacheKey(ctx context.Context, key string, ds *dataset.Dataset) (*dataset.Stats, error) {
	if key != "" {
		if r, err := s.cache.JSON(ctx, key); err == nil {
			// return r, nil
		}
	}
	return s.calcStats(ctx, key, ds)
}

func (s *Service) calcStats(ctx context.Context, cacheKey string, ds *dataset.Dataset) (*dataset.Stats, error) {
	body := ds.BodyFile()
	if body == nil {
		return nil, fmt.Errorf("stats: dataset has no body file")
	}
	if ds.Structure == nil {
		return nil, fmt.Errorf("stats: dataset is missing structure")
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

	values := dsstats.ToMap(acc)

	if cacheKey != "" {
		go func() {
			data, err := json.Marshal(values)
			if err != nil {
				return
			}
			if err = s.cache.PutJSON(context.Background(), cacheKey, bytes.NewReader(data)); err != nil {
				log.Debugf("putting stats in cache: %v", err.Error())
			}
		}()
	}

	return &dataset.Stats{Stats: values}, nil
}

// FileInfoCacheKey combines modTime and filepath to create a cache key string
func FileInfoCacheKey(fi os.FileInfo) string {
	return fmt.Sprintf("%d-%s", fi.ModTime(), fi.Name())
}
