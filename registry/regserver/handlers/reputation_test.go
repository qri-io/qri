package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qri-io/qri/registry"
)

func TestReputation(t *testing.T) {
	freshRep := &registry.Reputation{
		ProfileID: "freshRep",
		Rep:       1,
	}

	badRep := &registry.Reputation{
		ProfileID: "badRep",
		Rep:       -1,
	}

	goodRep := &registry.Reputation{
		ProfileID: "goodRep",
		Rep:       10,
	}

	memReps := registry.NewMemReputations()
	memReps.Add(badRep)
	memReps.Add(goodRep)

	if memReps.Len() != 2 {
		t.Errorf("error adding initial reps to in memory reputations list")
		return
	}

	s := httptest.NewServer(NewRoutes(registry.Registry{Reputations: memReps}))

	type env struct {
		Data *registry.ReputationResponse
		Meta struct {
			Code int
		}
	}

	cases := []struct {
		method      string
		endpoint    string
		contentType string
		profileID   string
		resStatus   int
		reputation  *registry.Reputation
	}{
		{"BAD_METHOD", "/registry/reputation", "", "", http.StatusBadRequest, nil},
		{"BAD_METHOD", "/registry/reputation", "application/json", "", http.StatusBadRequest, nil},
		{"BAD_METHOD", "/registry/reputation", "application/json", "my_id", http.StatusNotFound, nil},
		{"GET", "/registry/reputation", "", "", http.StatusBadRequest, nil},
		{"GET", "/registry/reputation", "application/json", "", http.StatusBadRequest, nil},
		{"GET", "/registry/reputation", "application/json", "freshRep", http.StatusOK, freshRep},
		{"GET", "/registry/reputation", "application/json", "badRep", http.StatusOK, badRep},
		{"GET", "/registry/reputation", "application/json", "goodRep", http.StatusOK, goodRep},
	}

	for i, c := range cases {
		requestRep := registry.NewReputation(c.profileID)
		req, err := http.NewRequest(c.method, fmt.Sprintf("%s%s", s.URL, c.endpoint), nil)
		if err != nil {
			t.Errorf("case %d error creating request: %s", i, err.Error())
			continue
		}

		if c.contentType != "" {
			req.Header.Set("Content-Type", c.contentType)
		}
		if c.profileID != "" {
			data, err := json.Marshal(requestRep)
			if err != nil {
				t.Errorf("error marshaling json body: %s", err.Error())
				return
			}
			req.Body = ioutil.NopCloser(bytes.NewReader([]byte(data)))
		}

		res, err := http.DefaultClient.Do(req)
		if res.StatusCode != c.resStatus {
			t.Errorf("case %d res status mismatch. expected: %d, got: %d", i, c.resStatus, res.StatusCode)
			continue
		}

		if c.reputation != nil {
			e := &env{}
			if err := json.NewDecoder(res.Body).Decode(e); err != nil {
				t.Errorf("case %d error reading response body: %s", i, err.Error())
				continue
			}

			res := e.Data
			rep := res.Reputation

			if c.reputation.ProfileID != rep.ProfileID {
				t.Errorf("case %d reputation profileID mismatch. expected: %s, got:%s", i, c.reputation.ProfileID, rep.ProfileID)
			}

			if c.reputation.Rep != rep.Rep {
				t.Errorf("case %d Reputation mismatch. expected: %d, got: %d", i, c.reputation.Rep, rep.Rep)
			}
		}
	}
}
