package stats

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/qri-io/dataset"
)

var (
	// ErrCacheMiss indicates a requested path isn't in the cache
	ErrCacheMiss = fmt.Errorf("stats: cache miss")
	// ErrNoCache indicates there is no cache
	ErrNoCache = fmt.Errorf("stats: no cache")
	// ErrCacheCorrupt indicates a faulty stats cache
	ErrCacheCorrupt = fmt.Errorf("stats: cache is corrupt")
)

// Cache is a store of stats components
// Consumers of a cache must not rely on the cache for persistence
// Implementations are expected to maintain their own size bounding
// semantics internally
// Cache implementations must be safe for concurrent use, and must be
// nil-callable
type Cache interface {
	// placing a stats object in the Cache will expire all caches with a lower
	// modTime, use a modTime of zero when no modTime is known
	PutStats(ctx context.Context, key string, modTime int, sa *dataset.Stats) error
	// get the cached modTime for
	// will return ErrCacheMiss if the key does not exist
	ModTime(ctx context.Context, key string) (modTime int, err error)
	// Get the stats component for a given key
	Stats(ctx context.Context, key string) (sa *dataset.Stats, err error)
}

// nilCache is a stand in for not having a cache
// it only ever returns ErrNoCache
type nilCache bool

var _ Cache = (*nilCache)(nil)

// PutJSON places stats in the cache, keyed by path
func (nilCache) PutStats(ctx context.Context, key string, modTime int, sa *dataset.Stats) error {
	return ErrNoCache
}

// ModTime always returns ErrCacheMiss
func (nilCache) ModTime(ctx context.Context, key string) (modTime int, err error) {
	return -1, ErrCacheMiss
}

// JSON gets cached byte data for a path
func (nilCache) Stats(ctx context.Context, key string) (sa *dataset.Stats, err error) {
	return nil, ErrCacheMiss
}

// osCache is a stats cache stored in a directory on the local operating system
type osCache struct {
	root    string
	maxSize uint64

	info   *cacheInfo
	infoLk sync.Mutex
}

var _ Cache = (*osCache)(nil)

// NewOSCache creates a cache in a local direcory
func NewOSCache(rootDir string, maxSize uint64) (Cache, error) {
	c := &osCache{
		root:    rootDir,
		maxSize: maxSize,
	}

	err := c.readCacheInfo()
	if errors.Is(err, ErrCacheCorrupt) {
		log.Warn("your cache of stats data is corrupt, removing all cached data")
		err = os.RemoveAll(rootDir)
		return c, err
	}

	if err := os.MkdirAll(rootDir, os.ModePerm); err != nil {
		return nil, err
	}

	return c, err
}

// Put places stats in the cache, keyed by path
func (c *osCache) PutStats(ctx context.Context, key string, modTime int, sa *dataset.Stats) error {
	if modTime < 0 {
		modTime = 0
	}

	key = c.cacheKey(key)
	filename := c.componentFilepath(key)
	data, err := json.Marshal(sa)
	if err != nil {
		return err
	}

	if uint64(len(data)) > c.maxSize {
		return fmt.Errorf("stats component size exceeds maximum size of cache")
	}

	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return err
	}

	c.addAndPurgeExcess(key, modTime, uint64(len(data)))

	return c.writeCacheInfo()
}

func (c *osCache) ModTime(ctx context.Context, key string) (modTime int, err error) {
	var exists bool
	if modTime, exists = c.info.ModTimes[c.cacheKey(key)]; !exists {
		return -1, ErrCacheMiss
	}

	return modTime, nil
}

// Stats gets cached byte data for a path
func (c *osCache) Stats(ctx context.Context, key string) (sa *dataset.Stats, err error) {
	return nil, ErrCacheMiss
}

var b32Enc = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

func (c *osCache) componentFilepath(cacheKey string) string {
	return filepath.Join(c.root, fmt.Sprintf("%s.json", cacheKey))
}

func (c *osCache) cacheKey(key string) string {
	return b32Enc.EncodeToString([]byte(key))
}

const uintSize = 32 << (^uint(0) >> 32 & 1)

func (c *osCache) addAndPurgeExcess(cacheKey string, modTime int, size uint64) {
	c.infoLk.Lock()
	defer c.infoLk.Unlock()
	c.info.Sizes[cacheKey] = uint64(len(data))
	c.info.ModTimes[cacheKey] = modTime

	var (
		lowestKey     string
		lowestModTime int
	)

	for c.info.Size() > c.maxSize {
		lowestKey = ""
		lowestModTime = 1<<(uintSize-1) - 1

		for key, modTime := range c.info.ModTimes {
			if modTime < lowestModTime {
				lowestKey = key
			}
		}
		if lowestKey == "" {
			break
		}
		if err := os.Remove(c.componentFilepath(lowestKey)); err != nil {
			break
		}
		delete(c.info.Sizes, lowestKey)
		delete(c.info.ModTimes, lowestKey)
	}
}

const osCacheInfoFilename = "info.json"

type cacheInfo struct {
	Sizes    map[string]uint64
	ModTimes map[string]int
}

func (ci cacheInfo) Size() (size uint64) {
	for _, s := range ci.Sizes {
		size += s
	}
	return size
}

func (c *osCache) readCacheInfo() error {
	c.infoLk.Lock()
	defer c.infoLk.Unlock()

	name := filepath.Join(c.root, osCacheInfoFilename)
	f, err := os.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			c.info = &cacheInfo{}
			return nil
		}
		return err
	}

	defer f.Close()

	c.info = &cacheInfo{}
	if err := json.NewDecoder(f).Decode(c.info); err != nil {
		// corrupt cache
		return fmt.Errorf("%w decoding stats info: %s", ErrCacheCorrupt, err)
	}

	return nil
}

func (c *osCache) writeCacheInfo() error {
	c.infoLk.Lock()
	defer c.infoLk.Unlock()

	name := filepath.Join(c.root, osCacheInfoFilename)
	data, err := json.Marshal(c.info)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(name, data, 0644)
}
