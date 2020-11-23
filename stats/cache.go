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
	"github.com/qri-io/didmod"
	"github.com/qri-io/qfs"
)

var (
	// ErrCacheMiss indicates a requested path isn't in the cache
	ErrCacheMiss = fmt.Errorf("stats: cache miss")
	// ErrNoCache indicates there is no cache
	ErrNoCache = fmt.Errorf("stats: no cache")
	// ErrCacheCorrupt indicates a faulty stats cache
	ErrCacheCorrupt = fmt.Errorf("stats: cache is corrupt")
)

// Props is an alias of
type Props = didmod.Props

// Cache is a store of stats components
// Consumers of a cache must not rely on the cache for persistence
// Implementations are expected to maintain their own size bounding
// semantics internally
// Cache implementations must be safe for concurrent use, and must be
// nil-callable
type Cache interface {
	// placing a stats object in the Cache will expire all caches with a lower
	// modTime, use a modTime of zero when no modTime is known
	PutStats(ctx context.Context, key string, sa *dataset.Stats) error
	// GetStats the stats component for a given key
	GetStats(ctx context.Context, key string) (sa *dataset.Stats, err error)
}

// nilCache is a stand in for not having a cache
// it only ever returns ErrNoCache
type nilCache bool

var _ Cache = (*nilCache)(nil)

// PutJSON places stats in the cache, keyed by path
func (nilCache) PutStats(ctx context.Context, key string, sa *dataset.Stats) error {
	return ErrNoCache
}

// JSON gets cached byte data for a path
func (nilCache) GetStats(ctx context.Context, key string) (sa *dataset.Stats, err error) {
	return nil, ErrCacheMiss
}

// localCache is a stats cache stored in a directory on the local operating system
type localCache struct {
	root    string
	maxSize int64

	info   *cacheInfo
	infoLk sync.Mutex
}

var _ Cache = (*localCache)(nil)

// NewLocalCache creates a cache in a local directory. LocalCache is sensitive
// to added keys that match the qfs.PathKind of "local". When a stats component
// is added with a local filepath as it's key, LocalCache will record the
// status of that file,  and return ErrCacheMiss if that filepath is altered on
// retrieval
func NewLocalCache(rootDir string, maxSize int64) (Cache, error) {
	c := &localCache{
		root:    rootDir,
		maxSize: maxSize,
		info:    newCacheInfo(),
	}

	err := c.readCacheInfo()
	if errors.Is(err, ErrCacheCorrupt) {
		log.Warn("Cache of stats data is corrupt. Removing all cached stats data as a precaution. This isn't too big a deal, as stats data can be recalculated.")
		err = os.RemoveAll(rootDir)
		return c, err
	}

	// ensure base directory exists
	if err := os.MkdirAll(rootDir, os.ModePerm); err != nil {
		return nil, err
	}

	return c, err
}

// Put places stats in the cache, keyed by path
func (c *localCache) PutStats(ctx context.Context, key string, sa *dataset.Stats) (err error) {
	var statProps, targetProps didmod.Props
	if qfs.PathKind(key) == "local" {
		targetProps, _ = didmod.NewProps(key)
	}

	key = c.cacheKey(key)
	filename := c.componentFilepath(key)
	data, err := json.Marshal(sa)
	if err != nil {
		return err
	}

	if int64(len(data)) > c.maxSize {
		return fmt.Errorf("stats component size exceeds maximum size of cache")
	}

	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return err
	}
	statProps, err = didmod.NewProps(filename)
	if err != nil {
		return err
	}

	c.addAndPurgeExpired(key, statProps, targetProps)

	return c.writeCacheInfo()
}

// Stats gets cached byte data for a path
func (c *localCache) GetStats(ctx context.Context, key string) (sa *dataset.Stats, err error) {
	cacheKey := c.cacheKey(key)
	log.Debugw("getting stats", "key", key, "cacheKey", cacheKey)

	targetFileProps, exists := c.info.TargetFileProps[cacheKey]
	if !exists {
		return nil, ErrCacheMiss
	}

	if qfs.PathKind(key) == "local" {
		if fileProps, err := didmod.NewProps(key); err == nil {
			if !targetFileProps.Equal(fileProps) {
				// note: returning ErrCacheMiss here will probably lead to re-calcualtion
				// and subsequent overwriting by cache consumers, so we shouldn't need
				// to proactively drop the stale cache here
				return nil, ErrCacheMiss
			}
		}
	}

	f, err := os.Open(c.componentFilepath(cacheKey))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sa = &dataset.Stats{}
	err = json.NewDecoder(f).Decode(sa)
	return sa, err
}

var b32Enc = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

func (c *localCache) componentFilepath(cacheKey string) string {
	return filepath.Join(c.root, fmt.Sprintf("%s.json", cacheKey))
}

func (c *localCache) cacheKey(key string) string {
	return b32Enc.EncodeToString([]byte(key))
}

const uintSize = 32 << (^uint(0) >> 32 & 1)

func (c *localCache) addAndPurgeExpired(cacheKey string, statProps, targetProps didmod.Props) {
	c.infoLk.Lock()
	defer c.infoLk.Unlock()
	log.Debugw("adding stat props", "cacheKey", cacheKey, "statProps", statProps, "targetProps", targetProps)
	c.info.StatFileProps[cacheKey] = statProps
	c.info.TargetFileProps[cacheKey] = targetProps

	var (
		lowestKey     string
		lowestModTime int
	)

	for c.info.Size() > c.maxSize {
		lowestKey = ""
		lowestModTime = 1<<(uintSize-1) - 1

		for key, fileProps := range c.info.StatFileProps {
			if int(fileProps.Mtime.Unix()) < lowestModTime && key != cacheKey {
				lowestKey = key
			}
		}
		if lowestKey == "" {
			break
		}
		log.Debugw("dropping stats component from local cache", "path", lowestKey, "size", c.info.StatFileProps[lowestKey].Size)
		if err := os.Remove(c.componentFilepath(lowestKey)); err != nil {
			break
		}
		delete(c.info.StatFileProps, lowestKey)
		delete(c.info.TargetFileProps, lowestKey)
	}
}

const localCacheInfoFilename = "info.json"

type cacheInfo struct {
	StatFileProps   map[string]didmod.Props
	TargetFileProps map[string]didmod.Props
}

func newCacheInfo() *cacheInfo {
	return &cacheInfo{
		StatFileProps:   map[string]didmod.Props{},
		TargetFileProps: map[string]didmod.Props{},
	}
}

func (ci cacheInfo) Size() (size int64) {
	for _, p := range ci.StatFileProps {
		size += p.Size
	}
	return size
}

func (c *localCache) readCacheInfo() error {
	c.infoLk.Lock()
	defer c.infoLk.Unlock()

	name := filepath.Join(c.root, localCacheInfoFilename)
	f, err := os.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			c.info = newCacheInfo()
			return nil
		}
		return err
	}

	defer f.Close()

	c.info = newCacheInfo()
	if err := json.NewDecoder(f).Decode(c.info); err != nil {
		// corrupt cache
		return fmt.Errorf("%w decoding stats info: %s", ErrCacheCorrupt, err)
	}

	return nil
}

func (c *localCache) writeCacheInfo() error {
	c.infoLk.Lock()
	defer c.infoLk.Unlock()

	name := filepath.Join(c.root, localCacheInfoFilename)
	data, err := json.Marshal(c.info)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(name, data, 0644)
}
