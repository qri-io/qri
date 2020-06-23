package config

import (
	"reflect"
	"testing"
)

func TestRepoValidate(t *testing.T) {
	err := DefaultRepo().Validate()
	if err != nil {
		t.Errorf("error validating default repo: %s", err)
	}
}

func TestRepoCopy(t *testing.T) {
	// build off DefaultRepo so we can test that the repo Copy
	// actually copies over correctly (ie, deeply)
	r := DefaultRepo()

	cases := []struct {
		repo *Repo
	}{
		{r},
	}
	for i, c := range cases {
		cpy := c.repo.Copy()
		if !reflect.DeepEqual(cpy, c.repo) {
			t.Errorf("Repo Copy test case %v, repo structs are not equal: \ncopy: %v, \noriginal: %v", i, cpy, c.repo)
			continue
		}
		cpy.Path = "newPath"
		if reflect.DeepEqual(cpy, c.repo) {
			t.Errorf("Repo Copy test case %v, editing one repo struct should not affect the other: \ncopy: %v, \noriginal: %v", i, cpy, c.repo)
			continue
		}
	}
}
