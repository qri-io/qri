package config

import (
	"fmt"
)

// Remotes encapsulates configuration options for remotes
type Remotes struct {
	nameMap map[string]string
}

// SetArbitrary is for implementing the ArbitrarySetter interface defined by base/fill_struct.go
func (r *Remotes) SetArbitrary(key string, val interface{}) (err error) {
	if r.nameMap == nil {
		r.nameMap = map[string]string{}
	}
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("invalid remote value: %s", val)
	}
	r.nameMap[key] = str
	return nil
}

// Get retrieves an address from the name of remote
func (r *Remotes) Get(name string) (string, bool) {
	addr, ok := r.nameMap[name]
	return addr, ok
}
