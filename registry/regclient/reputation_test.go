package regclient

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver/handlers"
)

func TestReputationRequests(t *testing.T) {
	profileID := "my_id"
	profileID2 := "my_second_id"
	memRs := registry.NewMemReputations()
	newRep := registry.NewReputation(profileID)
	newRep.SetReputation(-1)
	err := memRs.Add(newRep)
	if err != nil {
		t.Error(err)
	}

	reg := registry.Registry{
		Reputations: memRs,
	}
	ts := httptest.NewServer(handlers.NewRoutes(reg))
	c := NewClient(&Config{
		Location: ts.URL,
	})

	res, err := c.GetReputation(profileID)
	if err != nil {
		t.Error(err)
		return
	}

	rep := res.Reputation
	ttl := res.Expiration
	if -1 != rep.Reputation() {
		t.Errorf("reputation value not equal: expect -1, got %d\n", rep.Reputation())
	}

	if ttl != time.Hour*24 {
		t.Errorf("reputation expiration not equal: expect 24 hours got %d\n", ttl)
	}

	res, err = c.GetReputation(profileID2)
	if err != nil {
		t.Error(err)
		return
	}

	rep = res.Reputation
	ttl = res.Expiration
	if 1 != rep.Reputation() {
		t.Errorf("reputation value not equal: expect 1, got %d", rep.Reputation())
	}
	if ttl != time.Hour*24 {
		t.Errorf("reputation expiration not equal: expect 24 hours got %d\n", ttl)
	}

	if memRs.Len() != 2 {
		t.Errorf("reputations list should equal 2, got %d", memRs.Len())
	}
}
