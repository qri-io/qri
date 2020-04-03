package resolver

import (
	"testing"

	"github.com/qri-io/qri/dsref"
)

func TestMemResolver(t *testing.T) {
	m := NewMemResolver()
	m.Put(dsref.VersionInfo{
		InitID: "myInitID",
		Username: "test_peer",
		Name: "my_ds",
		Path: "/ipfs/QmeXaMpLe",
	})
	expectPath := "/ipfs/QmeXaMpLe"

	info := m.GetInfo("invalid")
	if info != nil {
		t.Errorf("expected info to be nil")
	}

	info = m.GetInfo("myInitID")
	if info == nil {
		t.Fatal("unexpected nil info")
	}
	if info.Path != expectPath {
		t.Errorf("path mismatch: expect %q, got %q", expectPath, info.Path)
	}

	info = m.GetInfoByDsref(dsref.Ref{Username: "test_peer", Name: "not_found"})
	if info != nil {
		t.Errorf("expected info to be nil")
	}

	info = m.GetInfoByDsref(dsref.Ref{Username: "test_peer", Name: "my_ds"})
	if info == nil {
		t.Fatal("unexpected nil info")
	}
	if info.Path != expectPath {
		t.Errorf("path mismatch: expect %q, got %q", expectPath, info.Path)
	}
}
