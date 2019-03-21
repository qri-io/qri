package config

import (
	"fmt"
)

// Remotes encapsulates configuration options for remotes
type Remotes map[string]string

// SetArbitrary is for implementing the ArbitrarySetter interface defined by base/fill_struct.go
func (r *Remotes) SetArbitrary(key string, val interface{}) (err error) {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("invalid remote value: %s", val)
	}
	(*r)[key] = str
	return nil
}

// Get retrieves an address from the name of remote
func (r *Remotes) Get(name string) (string, bool) {
	addr, ok := (*r)[name]
	return addr, ok
}

// Copy creates a copy of a Remotes struct
func (r *Remotes) Copy() *Remotes {
	c := make(map[string]string)
	for k, v := range *r {
		c[k] = v
	}
	return (*Remotes)(&c)
}
