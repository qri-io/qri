package stats

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"
)

func TestOSCache(t *testing.T) {
	t.Skip("skipping caching for now")

	tmp, err := ioutil.TempDir("", "test_os_cache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	ctx := context.Background()
	// create a cache with a  100 byte max size
	cache := NewOSCache(tmp, 100)

	// put data at path "statsA"
	statsA := bytes.Repeat([]byte{'o'}, 50)
	if err = cache.PutJSON(ctx, "statsA", bytes.NewReader(statsA)); err != nil {
		t.Errorf("expected putting json data to not fail. got: %s", err)
	}
	got := cacheBytes(t, cache, "statsA")
	if !bytes.Equal(statsA, got) {
		t.Errorf("response bytes don't match input bytes.")
	}

	// overwrite data at path "statsA"
	statsA2 := bytes.Repeat([]byte{'p'}, 50)
	if err = cache.PutJSON(ctx, "statsA", bytes.NewReader(statsA)); err != nil {
		t.Errorf("expected putting json data to not fail. got: %s", err)
	}
	got = cacheBytes(t, cache, "statsA")
	if !bytes.Equal(statsA2, got) {
		t.Errorf("expeted putting the same path to overwrite cache data. response bytes don't match.")
	}

	// add data that'll overflow the cache max size
	statsB := bytes.Repeat([]byte{'o'}, 51)
	if err = cache.PutJSON(ctx, "statsB", bytes.NewReader(statsB)); err != nil {
		t.Errorf("expected putting long json data to not fail. got: %s", err)
	}

	_, getAErr := cache.JSON(ctx, "statsA")
	_, getBErr := cache.JSON(ctx, "statsB")
	if getAErr != ErrCacheMiss && getBErr != ErrCacheMiss {
		t.Errorf("expected at least one cache in an overflow state to ErrCacheMiss. got:\n\tstatA: %v\n\tstatB: %v", getAErr, getBErr)
	}
}

func cacheBytes(t *testing.T, c Cache, path string) []byte {
	r, err := c.JSON(context.Background(), path)
	if err != nil {
		t.Fatalf("getting cache path '%s': %s", path, err)
	}
	got, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return got
}
