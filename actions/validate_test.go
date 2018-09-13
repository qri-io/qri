package actions

import "testing"

func TestValidate(t *testing.T) {
	node := newTestNode(t)
	cities := addCitiesDataset(t, node)

	errs, err := Validate(node, cities, nil, nil)
	if err != nil {
		t.Error(err.Error())
	}

	if len(errs) != 0 {
		t.Errorf("expected 0 errors. got: %d", len(errs))
	}
}
