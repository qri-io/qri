package key

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/gofrs/flock"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/qri/config"
)

// ErrKeyAndIDMismatch occurs when a key identifier doesn't match it's public
// key
var ErrKeyAndIDMismatch = fmt.Errorf("public key does not match identifier")

// Store is an abstraction over a KeyBook
// In the future we may expand this interface to store symmetric encryption keys
type Store interface {
	Book
}

// NewStore constructs a keys.Store backed by memory or local file
func NewStore(cfg *config.Config) (Store, error) {
	if cfg.Repo == nil {
		return NewMemStore()
	}

	switch cfg.Repo.Type {
	case "fs":
		// Don't create a localstore with the empty path, this will use the current directory
		if cfg.Path() == "" {
			return nil, fmt.Errorf("new key.LocalStore requires non-empty path")
		}
		return NewLocalStore(filepath.Join(filepath.Dir(cfg.Path()), "keystore.json"))
	case "mem":
		return NewMemStore()
	default:
		return nil, fmt.Errorf("unknown repo type: %s", cfg.Repo.Type)
	}
}

type memStore struct {
	Book
}

// NewMemStore constructs an in-memory key.Store
func NewMemStore() (Store, error) {
	return &memStore{
		Book: newKeyBook(),
	}, nil
}

type localStore struct {
	sync.Mutex
	filename string
	flock    *flock.Flock
}

// NewLocalStore constructs a local file backed key.Store
func NewLocalStore(filename string) (Store, error) {
	return &localStore{
		filename: filename,
		flock:    flock.New(lockPath(filename)),
	}, nil
}

func lockPath(filename string) string {
	return fmt.Sprintf("%s.lock", filename)
}

// PubKey returns the public key for a given ID if it exists
func (s *localStore) PubKey(ctx context.Context, keyID ID) crypto.PubKey {
	s.Lock()
	defer s.Unlock()

	kb, err := s.keys()
	if err != nil {
		return nil
	}
	return kb.PubKey(ctx, keyID)
}

// PrivKey returns the private key for a given ID if it exists
func (s *localStore) PrivKey(ctx context.Context, keyID ID) crypto.PrivKey {
	s.Lock()
	defer s.Unlock()

	kb, err := s.keys()
	if err != nil {
		return nil
	}
	return kb.PrivKey(ctx, keyID)
}

// AddPubKey inserts a public key for a given ID
func (s *localStore) AddPubKey(ctx context.Context, keyID ID, pubKey crypto.PubKey) error {
	s.Lock()
	defer s.Unlock()

	kb, err := s.keys()
	if err != nil {
		return err
	}
	if !keyID.MatchesPublicKey(pubKey) {
		return fmt.Errorf("%w id: %q", ErrKeyAndIDMismatch, keyID.Pretty())
	}
	err = kb.AddPubKey(ctx, keyID, pubKey)
	if err != nil {
		return err
	}

	return s.saveFile(kb)
}

// AddPrivKey inserts a private key for a given ID
func (s *localStore) AddPrivKey(ctx context.Context, keyID ID, privKey crypto.PrivKey) error {
	s.Lock()
	defer s.Unlock()

	if !keyID.MatchesPrivateKey(privKey) {
		return fmt.Errorf("%w id: %q", ErrKeyAndIDMismatch, keyID.Pretty())
	}

	kb, err := s.keys()
	if err != nil {
		return err
	}
	err = kb.AddPrivKey(ctx, keyID, privKey)
	if err != nil {
		return err
	}

	return s.saveFile(kb)
}

// IDsWithKeys returns the list of IDs in the KeyBook
func (s *localStore) IDsWithKeys(ctx context.Context) []ID {
	s.Lock()
	defer s.Unlock()

	kb, err := s.keys()
	if err != nil {
		// the keys method will safely return an empty list which we can use bellow
		log.Debugf("error loading peers with keys: %q", err.Error())
		return []ID{}
	}
	return kb.IDsWithKeys(ctx)
}

func (s *localStore) keys() (Book, error) {
	log.Debug("reading keys")

	if err := s.flock.Lock(); err != nil {
		return nil, err
	}
	defer func() {
		log.Debug("keys read")
		s.flock.Unlock()
	}()

	kb := newKeyBook()
	data, err := ioutil.ReadFile(s.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return kb, nil
		}
		log.Debug(err.Error())
		return kb, fmt.Errorf("error loading keys: %s", err.Error())
	}

	if err := json.Unmarshal(data, kb); err != nil {
		log.Error(err.Error())
		// on bad parsing we simply return an empty keybook
		return kb, nil
	}
	return kb, nil
}

func (s *localStore) saveFile(kb Book) error {
	data, err := json.Marshal(kb)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	log.Debugf("writing keys: %s", s.filename)
	if err := s.flock.Lock(); err != nil {
		return err
	}
	defer func() {
		s.flock.Unlock()
		log.Debug("keys written")
	}()
	return ioutil.WriteFile(s.filename, data, 0644)
}
