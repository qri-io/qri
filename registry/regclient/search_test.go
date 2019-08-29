package regclient

import (
	"net/http/httptest"
	"testing"

	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver/handlers"
)

func TestSearchMethods(t *testing.T) {

	reg := registry.Registry{
		Profiles: registry.NewMemProfiles(),
		Search:   registry.MockSearch{},
	}

	srv := httptest.NewServer(handlers.NewRoutes(reg))
	c := NewClient(&Config{
		Location: srv.URL,
	})

	searchParams := &SearchParams{QueryString: "presidents", Limit: 100, Offset: 0}
	// TODO: need to add tests that actually inspect the search results
	_, err := c.Search(searchParams)
	if err != nil {
		t.Errorf("error executing search: %s", err)
	}
}
