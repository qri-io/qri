package stats

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
)

func TestLocalCache(t *testing.T) {
	tmp, err := ioutil.TempDir("", "test_os_cache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	ctx := context.Background()
	// create a cache with a  100 byte max size
	cache, err := NewLocalCache(tmp, 100)
	if err != nil {
		t.Fatal(err)
	}

	// put data at path "statsA"
	statsA := &dataset.Stats{
		Qri: dataset.KindStats.String(),
		Stats: []interface{}{
			map[string]interface{}{"type": "numeric"},
			map[string]interface{}{"type": "blah"},
		},
	}
	if err = cache.PutStats(ctx, "/mem/statsA", statsA); err != nil {
		t.Errorf("expected putting json data to not fail. got: %s", err)
	}

	gotStatsA, err := cache.GetStats(ctx, "/mem/statsA")
	if err != nil {
		t.Errorf("unexpected error getting stats: %q", err)
	}
	if diff := cmp.Diff(statsA, gotStatsA); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}

	// overwrite data at path "statsA"
	statsA = &dataset.Stats{
		Qri: dataset.KindStats.String(),
		Stats: []interface{}{
			map[string]interface{}{"type": "numeric"},
			map[string]interface{}{"type": "blah"},
			map[string]interface{}{"type": "blah"},
			map[string]interface{}{"type": "blah"},
		},
	}
	if err = cache.PutStats(ctx, "/mem/statsA", statsA); err != nil {
		t.Errorf("expected putting json data to not fail. got: %s", err)
	}

	gotStatsA, err = cache.GetStats(ctx, "/mem/statsA")
	if err != nil {
		t.Errorf("unexpected error getting stats: %q", err)
	}
	if diff := cmp.Diff(statsA, gotStatsA); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}

	// add data that'll overflow the cache max size
	statsB := &dataset.Stats{
		Qri: dataset.KindStats.String(),
		Stats: []interface{}{
			// big value to overflow cache
			strings.Repeat("o", 70),
		},
	}

	if err = cache.PutStats(ctx, "/mem/statsB", statsB); err != nil {
		t.Errorf("expected putting long stats to not fail. got: %s", err)
	}

	// expect statsA to now ErrCacheMiss b/c it's been garbage-collected when
	// statsB exceeded the max size
	if _, err := cache.GetStats(ctx, "/mem/statsA"); err == nil {
		t.Errorf("expected error getting stats after max-size exceeded. got nil")
	}

	f, err := ioutil.TempFile("", "stats_cache_test_local_file")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	defer os.Remove(path)

	fileStats := &dataset.Stats{
		Stats: []interface{}{
			map[string]interface{}{
				"hello": "world",
			},
		},
	}

	if err = cache.PutStats(ctx, path, fileStats); err != nil {
		t.Errorf("putting local file stats: %q", err)
	}

	if _, err := cache.GetStats(ctx, path); err != nil {
		t.Errorf("expected local cached stats to exist. got error: %q", err)
	}

	if err := os.Chmod(path, 0621); err != nil {
		t.Fatal(err)
	}

	_, err = cache.GetStats(ctx, path)
	if !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected local cached stats to return ErrCacheMiss after local path file permissions change. got error: %q", err)
	}
}
