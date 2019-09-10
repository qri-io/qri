package fsi

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"testing"

	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

// TmpPaths holds temporary data to cleanup, and derived values used by tests.
type TmpPaths struct {
	homeDir   string
	firstDir  string
	secondDir string

	testRepo repo.Repo
}

// NewTmpPaths constructs a new TmpPaths object.
func NewTmpPaths() *TmpPaths {
	testRepo, err := testrepo.NewTestRepo()
	if err != nil {
		panic(err)
	}
	homeDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}

	firstDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	secondDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	return &TmpPaths{homeDir: homeDir, firstDir: firstDir, secondDir: secondDir, testRepo: testRepo}
}

// Close cleans up TmpPaths.
func (t *TmpPaths) Close() {
	os.RemoveAll(t.homeDir)
	os.RemoveAll(t.firstDir)
	os.RemoveAll(t.secondDir)
}

func TestCreateLink(t *testing.T) {
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	link, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	expect := `peer/test_ds`
	if link != expect {
		t.Errorf("error: link value, actual: %s, expect: %s", link, expect)
	}

	actual, _ := ioutil.ReadFile(filepath.Join(paths.firstDir, ".qri-ref"))
	if string(actual) != expect {
		t.Errorf("error: .qri-ref content, actual: %s, expect: %s", actual, expect)
	}

	links, err := fsi.LinkedRefs(0, 30)
	if len(links) != 1 {
		t.Errorf("error: wanted links of length 1, got %d", len(links))
	}

	ls := links[0]
	if ls.AliasString() != "peer/test_ds" {
		t.Errorf("error: links[0].Ref got %s", ls.AliasString())
	}
	if ls.FSIPath != paths.firstDir {
		t.Errorf("error: links[0].Path, actual: %s, expect: %s", ls.FSIPath, paths.firstDir)
	}
}

func TestCreateLinkTwice(t *testing.T) {
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}
	_, err = fsi.CreateLink(paths.secondDir, "me/test_second")
	if err != nil {
		t.Fatalf(err.Error())
	}

	actual, _ := ioutil.ReadFile(filepath.Join(paths.firstDir, ".qri-ref"))
	expect := `peer/test_ds`
	if string(actual) != expect {
		t.Errorf("error: .qri-ref content, actual: %s, expect: %s", actual, expect)
	}

	actual, _ = ioutil.ReadFile(filepath.Join(paths.secondDir, ".qri-ref"))
	expect = `peer/test_second`
	if string(actual) != expect {
		t.Errorf("error: .qri-ref content, actual: %s, expect: %s", actual, expect)
	}

	links, err := fsi.LinkedRefs(0, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 {
		t.Errorf("error: wanted links of length 2, got %d", len(links))
	}

	ls := links[0]
	expectAlias := "peer/test_ds"
	if ls.AliasString() != expectAlias {
		t.Errorf("error: links[0].AliasString expected: %s got %s", expectAlias, ls.AliasString())
	}
	if ls.FSIPath != paths.firstDir {
		t.Errorf("error: links[0].Path, actual: %s, expect: %s", ls.FSIPath, paths.firstDir)
	}

	ls = links[1]
	expectAlias = "peer/test_second"
	if ls.AliasString() != "peer/test_second" {
		t.Errorf("error: links[1].AliasString expected: %s got %s", expectAlias, ls.AliasString())
	}
	if ls.FSIPath != paths.secondDir {
		t.Errorf("error: links[1].Path, actual: %s, expect: %s", ls.FSIPath, paths.secondDir)
	}
}

func TestCreateLinkAlreadyLinked(t *testing.T) {
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}
	_, err = fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err == nil {
		t.Errorf("expected an error, did not get one")
		return
	}

	expect := fmt.Sprintf("'peer/test_ds' is already linked to %s", paths.firstDir)
	if err.Error() != expect {
		t.Errorf("error didn't match, actual:\n%s\nexpect:\n%s", err.Error(), expect)
	}
}

func TestCreateLinkAgainOnceQriRefRemoved(t *testing.T) {
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}
	// Remove the .qri-ref link file, then CreateLink again.
	os.Remove(filepath.Join(paths.firstDir, ".qri-ref"))
	_, err = fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	actual, _ := ioutil.ReadFile(filepath.Join(paths.firstDir, ".qri-ref"))
	expect := `peer/test_ds`
	if string(actual) != expect {
		t.Errorf("error: .qri-ref content, actual: %s, expect: %s", actual, expect)
	}

	links, err := fsi.LinkedRefs(0, 30)
	if err != nil {
		t.Fatal(err)
	}

	if len(links) != 1 {
		t.Errorf("error: wanted links of length 1, got %d", len(links))
	}

	ls := links[0]
	expect = "peer/test_ds"
	if ls.AliasString() != expect {
		t.Errorf("error: links[0].AliasString expected: %s got %s", expect, ls.AliasString())
	}
	if ls.FSIPath != paths.firstDir {
		t.Errorf("error: links[0].Path, actual: %s, expect: %s", ls.FSIPath, paths.firstDir)
	}
}

func TestUpdateLink(t *testing.T) {
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}
	link, err := fsi.UpdateLink(paths.firstDir, "me/test_ds@/ipfs/QmExample")
	if err != nil {
		t.Fatalf(err.Error())
	}
	expect := `peer/test_ds@QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt/ipfs/QmExample`
	if link != expect {
		t.Errorf("error: link did not match, actual: %s, expect: %s", link, expect)
	}
}

func TestUnlink(t *testing.T) {
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo)
	_, err := fsi.CreateLink(paths.firstDir, "me/test_ds")
	if err != nil {
		t.Fatalf(err.Error())
	}

	if err := fsi.Unlink(paths.firstDir, "me/mismatched_reference"); err == nil {
		t.Errorf("expected unlinking mismatched reference to error")
	}

	if err := fsi.Unlink(paths.firstDir, "me/test_ds"); err != nil {
		t.Errorf("unlinking valid reference: %s", err.Error())
	}
}
