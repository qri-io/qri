package dsfs

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qfs"
	testkeys "github.com/qri-io/qri/auth/key/test"
	"github.com/qri-io/qri/event"
)

func TestLoadTransform(t *testing.T) {
	// TODO - restore
	// store := cafs.NewMapstore()
	// q := &dataset.AbstractTransform{Statement: "select * from whatever booooooo go home"}
	// a, err := SaveAbstractTransform(store, q, true)
	// if err != nil {
	// 	t.Errorf(err.Error())
	// 	return
	// }

	// if _, err = LoadTransform(store, a); err != nil {
	// 	t.Errorf(err.Error())
	// }
	// TODO - other tests & stuff
}

func TestLoadTransformScript(t *testing.T) {
	ctx := context.Background()
	fs := qfs.NewMemFS()

	privKey := testkeys.GetKeyData(10).PrivKey

	_, err := LoadTransformScript(ctx, fs, "")
	if err == nil {
		t.Error("expected load empty key to fail")
	}

	tc, err := dstest.NewTestCaseFromDir("testdata/cities_no_commit_title")
	if err != nil {
		t.Fatal(err.Error())
	}

	path, err := CreateDataset(ctx, fs, fs, event.NilBus, tc.Input, nil, privKey, SaveSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err.Error())
	}

	if _, err = LoadTransformScript(ctx, fs, path); err != ErrNoTransform {
		t.Errorf("expected no transform script error. got: %s", err)
	}

	tc, err = dstest.NewTestCaseFromDir("testdata/all_fields")
	if err != nil {
		t.Fatal(err.Error())
	}
	tsf, _ := tc.TransformScriptFile()
	transformPath, err := fs.Put(ctx, tsf)
	if err != nil {
		t.Fatal(err.Error())
	}
	tc.Input.Transform.ScriptPath = transformPath
	path, err = CreateDataset(ctx, fs, fs, event.NilBus, tc.Input, nil, privKey, SaveSwitches{Pin: true, ShouldRender: true})
	if err != nil {
		t.Fatal(err.Error())
	}

	file, err := LoadTransformScript(ctx, fs, path)
	if err != nil {
		t.Fatalf("expected transform script to load. got: %s", err)
	}

	tsf, _ = tc.TransformScriptFile()

	r := &EqualReader{file, tsf}
	if _, err := ioutil.ReadAll(r); err != nil {
		t.Error(err.Error())
	}
}

var ErrStreamsNotEqual = fmt.Errorf("streams are not equal")

// EqualReader confirms two readers are exactly the same, throwing an error
// if they return
type EqualReader struct {
	a, b io.Reader
}

func (r *EqualReader) Read(p []byte) (int, error) {
	pb := make([]byte, len(p))
	readA, err := r.a.Read(p)
	if err != nil {
		return readA, err
	}

	readB, err := r.b.Read(pb)
	if err != nil {
		return readA, err
	}

	if readA != readB {
		return readA, ErrStreamsNotEqual
	}

	for i, b := range p {
		if pb[i] != b {
			return readA, ErrStreamsNotEqual
		}
	}

	return readA, nil
}
