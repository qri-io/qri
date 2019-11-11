package stats

import (
	"context"
	"encoding/base32"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	// ErrCacheMiss indicates a requested path isn't in the cache
	ErrCacheMiss = fmt.Errorf("stats: cache miss")
	// ErrNoCache indicates there is no cache
	ErrNoCache = fmt.Errorf("stats: no cache")
)

// Cache is a store of JSON-formated stats data, keyed by path
// Consumers of a cache must not rely on the cache for persistence
// Implementations are expected to maintain their own size bounding
// semantics internally
// Cache implementations must be safe for concurrent use, and must be
// nil-callable
type Cache interface {
	// Put places stats in the cache, keyed by path
	PutJSON(ctx context.Context, path string, r io.Reader) error
	// JSON gets cached byte data for a path
	JSON(ctx context.Context, path string) (r io.Reader, err error)
}

// osCache is a stats cache stored in a directory on the local operating system
type osCache struct {
	root    string
	maxSize uint64
}

var _ Cache = (*osCache)(nil)

// NewOSCache creates a cache in a local direcory
func NewOSCache(rootDir string, maxSize uint64) Cache {
	if err := os.MkdirAll(rootDir, 0x667); err != nil {
		// log.Errorf("stat: %s", args ...interface{})
	}
	return osCache{
		root:    rootDir,
		maxSize: maxSize,
	}
}

// Put places stats in the cache, keyed by path
func (c osCache) PutJSON(ctx context.Context, path string, r io.Reader) error {
	filename := fmt.Sprintf("%s.json", b32Enc.EncodeToString([]byte(path)))
	// TODO (b5) - use this
	_ = filepath.Join(c.root, filename)

	return fmt.Errorf("not finished")
}

// JSON gets cached byte data for a path
func (c osCache) JSON(ctx context.Context, path string) (r io.Reader, err error) {
	return nil, ErrCacheMiss
}

var b32Enc = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

// nilCache is a stand in for not having a cache
// it only ever returns ErrNoCache
type nilCache bool

var _ Cache = (*nilCache)(nil)

// PutJSON places stats in the cache, keyed by path
func (nilCache) PutJSON(ctx context.Context, path string, r io.Reader) error {
	return ErrNoCache
}

// JSON gets cached byte data for a path
func (nilCache) JSON(ctx context.Context, path string) (r io.Reader, err error) {
	return nil, ErrNoCache
}
