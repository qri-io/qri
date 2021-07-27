package regclient

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver/handlers"
)

func TestSearchMethods(t *testing.T) {
	ctx := context.Background()

	reg := registry.Registry{
		Profiles: registry.NewMemProfiles(),
		Search:   registry.MockSearch{},
	}

	srv := httptest.NewServer(handlers.NewRoutes(reg))
	c := NewClient(&Config{
		Location: srv.URL,
	})

	searchParams := &SearchParams{Query: "presidents", Limit: 100, Offset: 0}
	// TODO: need to add tests that actually inspect the search results
	_, err := c.Search(ctx, searchParams)
	if err != nil {
		t.Errorf("error executing search: %s", err)
	}
}
