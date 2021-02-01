package fsi

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/repo"
	testrepo "github.com/qri-io/qri/repo/test"
)

func TestFilesystemPathToLocal(t *testing.T) {
	cases := []struct {
		in, expect string
	}{
		{"/path/to/local", "/path/to/local"},
		{"/fsi/path/to/local", "/path/to/local"},
		{"/fsi/fsi/local", "/fsi/local"},
	}

	for _, c := range cases {
		got := FilesystemPathToLocal(c.in)
		if c.expect != got {
			t.Errorf("result mismatch\nwant:\t%q\ngot:\t%s", c.expect, got)
		}
	}
}

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
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo, nil)
	vi, _, err := fsi.CreateLink(ctx, paths.firstDir, dsref.MustParse("peer/movies"))
	if err != nil {
		t.Fatalf(err.Error())
	}

	expect := &dsref.VersionInfo{
		Username:  "peer",
		Name:      "movies",
		ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		// TODO(b5) - NewTestRepo doesn't create stable path values
		// Path: "/map/shouldBeConstantForTests",
	}
	if diff := cmp.Diff(expect, vi, cmpopts.IgnoreFields(dsref.VersionInfo{}, "FSIPath", "Path")); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
	if vi.FSIPath == "" {
		t.Errorf("FSIPath cannot be empty")
	}

	actual, _ := ioutil.ReadFile(filepath.Join(paths.firstDir, ".qri-ref"))
	expectLinkFile := "peer/movies"
	if string(actual) != expectLinkFile {
		t.Errorf("wrong .qri-ref content.\nactual:\t%s\nexpect:\t%s", actual, expectLinkFile)
	}

	links, err := fsi.ListLinks(0, 30)
	if len(links) != 1 {
		t.Errorf("error: wanted links of length 1, got %d", len(links))
	}

	ls := links[0]
	if ls.SimpleRef().Human() != "peer/movies" {
		t.Errorf("error: links[0].Ref got %s", ls.SimpleRef().Human())
	}
	if ls.FSIPath != paths.firstDir {
		t.Errorf("error: links[0].Path, actual: %s, expect: %s", ls.FSIPath, paths.firstDir)
	}
}

func TestResolvedRef(t *testing.T) {
	t.Skip("TODO(b5) - owes a test")
}

func TestCreateLinkTwice(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo, nil)
	_, _, err := fsi.CreateLink(ctx, paths.firstDir, dsref.MustParse("peer/cities"))
	if err != nil {
		t.Fatalf(err.Error())
	}
	_, _, err = fsi.CreateLink(ctx, paths.secondDir, dsref.MustParse("peer/movies"))
	if err != nil {
		t.Fatalf(err.Error())
	}

	actual, _ := ioutil.ReadFile(filepath.Join(paths.firstDir, ".qri-ref"))
	expect := `peer/cities`
	if string(actual) != expect {
		t.Errorf("error: .qri-ref content, actual: %s, expect: %s", actual, expect)
	}

	actual, _ = ioutil.ReadFile(filepath.Join(paths.secondDir, ".qri-ref"))
	expect = `peer/movies`
	if string(actual) != expect {
		t.Errorf("error: .qri-ref content, actual: %s, expect: %s", actual, expect)
	}

	links, err := fsi.ListLinks(0, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 {
		t.Errorf("error: wanted links of length 2, got %d", len(links))
	}

	ls := links[0]
	expectAlias := "peer/cities"
	if ls.SimpleRef().Human() != expectAlias {
		t.Errorf("error: links[0].SimpleRef.Human() expected: %s got %s", expectAlias, ls.SimpleRef().Human())
	}
	if ls.FSIPath != paths.firstDir {
		t.Errorf("error: links[0].Path, actual: %s, expect: %s", ls.FSIPath, paths.firstDir)
	}

	ls = links[1]
	expectAlias = "peer/movies"
	if ls.SimpleRef().Human() != expectAlias {
		t.Errorf("error: links[1].SimpleRef.Human() expected: %s got %s", expectAlias, ls.SimpleRef().Human())
	}
	if ls.FSIPath != paths.secondDir {
		t.Errorf("error: links[1].Path, actual: %s, expect: %s", ls.FSIPath, paths.secondDir)
	}
}

func TestCreateLinkAlreadyLinked(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo, nil)
	_, _, err := fsi.CreateLink(ctx, paths.firstDir, dsref.MustParse("peer/cities"))
	if err != nil {
		t.Fatalf(err.Error())
	}
	_, _, err = fsi.CreateLink(ctx, paths.firstDir, dsref.MustParse("peer/cities"))
	if err == nil {
		t.Errorf("expected an error, did not get one")
		return
	}

	expect := fmt.Sprintf(`"peer/cities" is already linked to %s`, paths.firstDir)
	if err.Error() != expect {
		t.Errorf("error didn't match, actual:\n%s\nexpect:\n%s", err.Error(), expect)
	}
}

func TestCreateLinkAgainOnceQriRefRemoved(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo, nil)
	_, _, err := fsi.CreateLink(ctx, paths.firstDir, dsref.MustParse("peer/cities"))
	if err != nil {
		t.Fatalf(err.Error())
	}
	// Remove the .qri-ref link file, then CreateLink again.
	os.Remove(filepath.Join(paths.firstDir, ".qri-ref"))
	_, _, err = fsi.CreateLink(ctx, paths.firstDir, dsref.MustParse("peer/cities"))
	if err != nil {
		t.Fatalf(err.Error())
	}

	actual, _ := ioutil.ReadFile(filepath.Join(paths.firstDir, ".qri-ref"))
	expect := `peer/cities`
	if string(actual) != expect {
		t.Errorf("error: .qri-ref content, actual: %s, expect: %s", actual, expect)
	}

	links, err := fsi.ListLinks(0, 30)
	if err != nil {
		t.Fatal(err)
	}

	if len(links) != 1 {
		t.Errorf("error: wanted links of length 1, got %d", len(links))
	}

	ls := links[0]
	expect = "peer/cities"
	if ls.SimpleRef().Human() != expect {
		t.Errorf("error: links[0].SimpleRef.Human() expected: %s got %s", expect, ls.SimpleRef().Human())
	}
	if ls.FSIPath != paths.firstDir {
		t.Errorf("error: links[0].Path, actual: %s, expect: %s", ls.FSIPath, paths.firstDir)
	}
}

// Test that ModifyLinkReference changes what is in the .qri-ref linkfile
func TestModifyLinkReference(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo, nil)
	_, _, err := fsi.CreateLink(ctx, paths.firstDir, dsref.MustParse("peer/cities"))
	if err != nil {
		t.Fatal(err)
	}

	// TODO(dlong): This demonstrates a problem with how FSI is structured. The above call to
	// fsi.CreateLink will add the ref to the repo if it doesn't already exist. It also writes
	// to the linkfile (.qri-ref). The below call to ModifyLinkReference will modify the linkfile,
	// but it fails if the ref does not exist in the repo. The relationship between fsi and repo
	// is not clear and inconsistent.
	// UPDATE(b5): We need tests to confirm this, but requiring a dataset to exist before
	// allowing CreateLink makes qri the authoritative source of dataset data
	ref := dsref.MustParse("peer/cities")

	vi, err := repo.GetVersionInfoShim(paths.testRepo, ref)
	if err != nil {
		t.Fatal(err)
	}

	vi.Name = "cities_2"
	err = repo.PutVersionInfoShim(ctx, paths.testRepo, vi)
	if err != nil {
		t.Fatal(err)
	}

	// Modify the linkfile.
	_, err = fsi.ModifyLinkReference(paths.firstDir, dsref.MustParse("peer/cities_2"))
	if err != nil {
		t.Errorf("expected ModifyLinkReference to succeed, got: %s", err.Error())
	}

	// Verify that the working directory is linked to the expect dataset reference.
	ref, ok := GetLinkedFilesysRef(paths.firstDir)
	if !ok {
		t.Fatal("expected linked filesys ref, didn't get one")
	}
	expect := "peer/cities_2"
	if ref.Human() != expect {
		t.Errorf("expected %s, got %s", expect, ref.Human())
	}
}

// Test that ModifyLinkDirectory changes the FSIPath in the repo
func TestModifyLinkDirectory(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo, nil)
	_, _, err := fsi.CreateLink(ctx, paths.firstDir, dsref.MustParse("peer/movies"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = fsi.ModifyLinkDirectory(ctx, paths.secondDir, dsref.MustParse("peer/movies"))
	if err != nil {
		t.Errorf("expected ModifyLinkReference to succeed, got: %s", err.Error())
	}

	refs, err := fsi.ListLinks(0, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 1 {
		t.Fatalf("expected, 1 reference, got %d", len(refs))
	}

	actual := refs[0]
	expect := dsref.VersionInfo{
		Username:  "peer",
		Name:      "movies",
		ProfileID: "QmZePf5LeXow3RW5U1AgEiNbW46YnRGhZ7HPvm1UmPFPwt",
		FSIPath:   paths.secondDir,
	}
	if diff := cmp.Diff(expect, actual, cmpopts.IgnoreFields(dsref.VersionInfo{}, "FSIPath", "Path")); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
	if actual.FSIPath == "" {
		t.Errorf("FSIPath cannot be empty")
	}
}

func TestUnlink(t *testing.T) {
	ctx := context.Background()
	paths := NewTmpPaths()
	defer paths.Close()

	fsi := NewFSI(paths.testRepo, nil)
	_, _, err := fsi.CreateLink(ctx, paths.firstDir, dsref.MustParse("peer/cities"))
	if err != nil {
		t.Fatalf(err.Error())
	}

	if err := fsi.Unlink(ctx, paths.firstDir, dsref.MustParse("peer/mismatched_reference")); err == nil {
		t.Errorf("expected unlinking mismatched reference to error")
	}

	if err := fsi.Unlink(ctx, paths.firstDir, dsref.MustParse("peer/cities")); err != nil {
		t.Errorf("unlinking valid reference: %s", err.Error())
	}
}
