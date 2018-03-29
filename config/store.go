package config

// Store configures a qri content addessed file store (cafs)
type Store struct {
	Type string
}

// Default returns a new default Store configuration
func (Store) Default() *Store {
	return &Store{
		Type: "ipfs",
	}
}
