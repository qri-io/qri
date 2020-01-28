package lib

import (
	"context"
	"fmt"

	"github.com/qri-io/dataset"
)

// BrowseMethods enapsulates functions for surveying available datasets on the
// network.
type BrowseMethods struct {
	inst *Instance
}

// NewBrowseMethods creates a feed instance
func NewBrowseMethods(inst *Instance) *BrowseMethods {
	m := &BrowseMethods{
		inst: inst,
	}

	return m
}

// CoreRequestsName specifies this is a Methods object
func (m *BrowseMethods) CoreRequestsName() string {
	return "browse"
}

// Home returns a listing of datasets from a number of feeds like featured and
// popular. Each feed is keyed by string in the response
func (m *BrowseMethods) Home(p *bool, res *map[string][]*dataset.Dataset) error {
	if m.inst.rpc != nil {
		return m.inst.rpc.Call("BrowseMethods.Home", p, res)
	}
	ctx := context.TODO()

	if m.inst.registry == nil {
		return fmt.Errorf("Feed isn't available without a configured registry")
	}

	feed, err := m.inst.registry.HomeFeed(ctx)
	if err != nil {
		return err
	}

	*res = feed
	return nil
}

// Featured asks a registry for a curated list of datasets
func (m *BrowseMethods) Featured(p *ListParams, res *[]*dataset.Dataset) error {
	return fmt.Errorf("featured dataset feed is not yet implemented")
}

// Recent is a feed of network datasets in reverse chronological order
// it currently can only come from a registry, but could easily be assembled
// via p2p methods
func (m *BrowseMethods) Recent(p *ListParams, res *[]*dataset.Dataset) error {
	return fmt.Errorf("recent dataset feed is not yet implemented")
}
