package feature

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/gofrs/flock"
	"github.com/qri-io/qri/config"
)

// Store manages & stores feature flags
type Store interface {
	Lister
	// Get fetches a Flag from the Store using the ID
	Get(ctx context.Context, fid ID) (*Flag, error)
	// Put updates the flag user settings
	Put(ctx context.Context, f *Flag) (*Flag, error)
}

// A Lister lists entries from a feature flag store
type Lister interface {
	// List lists the Flags in the Store
	List(ctx context.Context) ([]*Flag, error)
}

// NewStore constructs a feature.Store backed by memory or local file
func NewStore(cfg *config.Config) (Store, error) {
	if cfg.Repo == nil {
		return NewMemStore(), nil
	}

	switch cfg.Repo.Type {
	case "fs":
		// Don't create a localstore with the empty path, this will use the current directory
		if cfg.Path() == "" {
			return nil, fmt.Errorf("new feature.LocalStore requires non-empty path")
		}
		return NewLocalStore(filepath.Join(filepath.Dir(cfg.Path()), "feature_flags.json"))
	case "mem":
		return NewMemStore(), nil
	default:
		return nil, fmt.Errorf("unknown repo type: %s", cfg.Repo.Type)
	}
}

// MemStore is an in memory representation of a Store
type MemStore struct {
	sync.Mutex
	flags     map[ID]*Flag
	overrides map[ID]*Flag
}

var _ Store = (*MemStore)(nil)

// NewMemStore returns a MemStore
func NewMemStore() *MemStore {
	return &MemStore{
		flags:     DefaultFlags,
		overrides: map[ID]*Flag{},
	}
}

// List lists the Flags in the Store
func (s *MemStore) List(ctx context.Context) ([]*Flag, error) {
	s.Lock()
	defer s.Unlock()

	flagMap := map[ID]*Flag{}
	for _, f := range s.flags {
		flagMap[f.ID] = f
	}
	for _, f := range s.overrides {
		flagMap[f.ID] = f
	}
	flags := []*Flag{}
	for _, f := range flagMap {
		flags = append(flags, f)
	}
	return flags, nil
}

// Get fetches a Flag from the Store using the ID
func (s *MemStore) Get(ctx context.Context, fid ID) (*Flag, error) {
	s.Lock()
	defer s.Unlock()

	if flag, ok := s.overrides[fid]; ok {
		return flag, nil
	}
	if flag, ok := s.flags[fid]; ok {
		return flag, nil
	}
	return &Flag{
		ID:     fid,
		Active: false,
	}, nil
}

// Put updates the flag user settings
func (s *MemStore) Put(ctx context.Context, f *Flag) (*Flag, error) {
	s.Lock()
	defer s.Unlock()

	s.overrides[f.ID] = f
	return f, nil
}

// LocalStore is a file backed representation of a Store
type LocalStore struct {
	sync.Mutex
	flags     map[ID]*Flag
	overrides map[ID]*Flag
	filename  string
	flock     *flock.Flock
}

var _ Store = (*LocalStore)(nil)

// NewLocalStore returns a LocalStore
func NewLocalStore(filename string) (*LocalStore, error) {
	store := &LocalStore{
		flags:     DefaultFlags,
		overrides: map[ID]*Flag{},
		filename:  filename,
		flock:     flock.New(fmt.Sprintf("%s.lock", filename)),
	}
	err := store.load()
	if err != nil {
		return nil, err
	}
	return store, nil
}

// List lists the Flags in the Store
func (s *LocalStore) List(ctx context.Context) ([]*Flag, error) {
	s.Lock()
	defer s.Unlock()

	flagMap := map[ID]*Flag{}
	for _, f := range s.flags {
		flagMap[f.ID] = f
	}
	for _, f := range s.overrides {
		flagMap[f.ID] = f
	}
	flags := []*Flag{}
	for _, f := range flagMap {
		flags = append(flags, f)
	}
	return flags, nil
}

// Get fetches a Flag from the Store using the ID
func (s *LocalStore) Get(ctx context.Context, fid ID) (*Flag, error) {
	s.Lock()
	defer s.Unlock()

	if flag, ok := s.overrides[fid]; ok {
		return flag, nil
	}
	if flag, ok := s.flags[fid]; ok {
		return flag, nil
	}
	return &Flag{
		ID:     fid,
		Active: false,
	}, nil
}

// Put updates the flag user settings
func (s *LocalStore) Put(ctx context.Context, f *Flag) (*Flag, error) {
	s.Lock()
	defer s.Unlock()

	s.overrides[f.ID] = f
	err := s.save()
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (s *LocalStore) load() error {
	if err := s.flock.Lock(); err != nil {
		return err
	}
	defer func() {
		s.flock.Unlock()
	}()

	flags := []*Flag{}

	data, err := ioutil.ReadFile(s.filename)
	if err != nil {
		if os.IsNotExist(err) {
			// on missing flag store we just continue normally
			return nil
		}
		return fmt.Errorf("error loading flags: %s", err.Error())
	}

	if err := json.Unmarshal(data, &flags); err != nil {
		return fmt.Errorf("error parsing flags: %s", err.Error())
	}

	for _, f := range flags {
		s.overrides[f.ID] = f
	}
	return nil
}

func (s *LocalStore) save() error {
	flags := []*Flag{}
	for _, f := range s.overrides {
		flags = append(flags, f)
	}
	data, err := json.Marshal(flags)
	if err != nil {
		return err
	}

	if err := s.flock.Lock(); err != nil {
		return err
	}
	defer func() {
		s.flock.Unlock()
	}()
	return ioutil.WriteFile(s.filename, data, 0644)
}
