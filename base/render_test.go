package base

import (
	"testing"
)

func TestRender(t *testing.T) {
	r := newTestRepo(t)
	ref := addCitiesDataset(t, r)

	_, err := Render(r, ref, nil, 0, 0, true)
	if err != nil {
		t.Error(err.Error())
	}

}
