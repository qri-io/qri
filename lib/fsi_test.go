package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
	cmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/p2p"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestFSIMethodsWrite(t *testing.T) {
	mr, err := testrepo.NewTestRepo()
	if err != nil {
		t.Fatalf("error allocating test repo: %s", err.Error())
	}
	node, err := p2p.NewQriNode(mr, config.DefaultP2PForTesting())
	if err != nil {
		t.Fatal(err.Error())
	}

	inst := NewInstanceFromConfigAndNode(config.DefaultConfigForTesting(), node)

	// we need some fsi stuff to fully test remove
	methods := NewFSIMethods(inst)
	// create datasets working directory
	datasetsDir, err := ioutil.TempDir("", "QriTestDatasetRequestsRemove")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(datasetsDir)

	// initialize an example no-history dataset
	initp := &InitFSIDatasetParams{
		Name:   "no_history",
		Dir:    datasetsDir,
		Format: "csv",
		Mkdir:  "no_history",
	}
	var noHistoryName string
	if err := methods.InitDataset(initp, &noHistoryName); err != nil {
		t.Fatal(err)
	}

	// link cities dataset with a checkout
	checkoutp := &CheckoutParams{
		Dir: filepath.Join(datasetsDir, "cities"),
		Ref: "me/cities",
	}
	var out string
	if err := methods.Checkout(checkoutp, &out); err != nil {
		t.Fatal(err)
	}

	// link craigslist with a checkout
	checkoutp = &CheckoutParams{
		Dir: filepath.Join(datasetsDir, "craigslist"),
		Ref: "me/craigslist",
	}
	if err := methods.Checkout(checkoutp, &out); err != nil {
		t.Fatal(err)
	}

	badCases := []struct {
		err    string
		params FSIWriteParams
	}{
		{"empty reference", FSIWriteParams{Ref: ""}},
		{"dataset name may not contain any upper-case letters", FSIWriteParams{Ref: "abc/ABC"}},
		{"dataset is required", FSIWriteParams{Ref: "abc/movies"}},
		{"unexpected character at position 0: '\xc3\xb0'", FSIWriteParams{Ref: "👋", Ds: &dataset.Dataset{}}},
		{"repo: not found", FSIWriteParams{Ref: "abc/movies", Ds: &dataset.Dataset{}}},
		{"dataset is not linked to the filesystem", FSIWriteParams{Ref: "peer/movies", Ds: &dataset.Dataset{}}},
	}

	for _, c := range badCases {
		t.Run(fmt.Sprintf("bad_case_%s", c.err), func(t *testing.T) {
			res := []StatusItem{}
			err := methods.Write(&c.params, &res)

			if err == nil {
				t.Errorf("expected error. got nil")
				return
			} else if c.err != err.Error() {
				t.Errorf("error mismatch: expected: %s, got: %s", c.err, err)
			}
		})
	}

	goodCases := []struct {
		description string
		params      FSIWriteParams
		res         []StatusItem
	}{
		{"update cities structure",
			FSIWriteParams{Ref: "me/cities", Ds: &dataset.Dataset{Structure: &dataset.Structure{Format: "json"}}},
			[]StatusItem{
				{Component: "meta", Type: "unmodified"},
				{Component: "structure", Type: "modified"},
				{Component: "body", Type: "unmodified"},
			},
		},
		// TODO (b5) - doesn't work yet
		// {"overwrite craigslist body",
		// 	FSIWriteParams{Ref: "me/craigslist", Ds: &dataset.Dataset{Body: []interface{}{[]interface{}{"foo", "bar", "baz"}}}},
		// 	[]StatusItem{},
		// },
		{"set title for no history dataset",
			FSIWriteParams{Ref: noHistoryName, Ds: &dataset.Dataset{Meta: &dataset.Meta{Title: "Changed Title"}}},
			[]StatusItem{
				{Component: "meta", Type: "add"},
				{Component: "structure", Type: "add"},
				{Component: "body", Type: "add"},
			},
		},
	}

	for _, c := range goodCases {
		t.Run(fmt.Sprintf("good_case_%s", c.description), func(t *testing.T) {
			res := []StatusItem{}
			err := methods.Write(&c.params, &res)

			if err != nil {
				t.Errorf("unexpected error: %s", err)
				return
			}

			if diff := cmp.Diff(c.res, res, cmpopts.IgnoreFields(StatusItem{}, "Mtime", "SourceFile")); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
