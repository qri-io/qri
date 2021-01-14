package preview

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dstest"
	"github.com/qri-io/qri/base/dsfs"
)

func TestCreatePreview(t *testing.T) {

	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return new(time.Time).In(time.UTC) }
	defer func() { dsfs.Timestamp = prevTs }()

	r := newTestRepo(t)
	turnstileRef := addTurnstileDataset(t, r)
	ctx := context.Background()

	got, err := CreatePreview(ctx, r.Filesystem(), turnstileRef)
	if err != nil {
		t.Fatal(err)
	}

	expect := dstest.LoadGoldenFile(t, "testdata/expect/TestCreatePreview.turnstile.json")
	// TODO (b5) - required adjustments for accurate comparison due to JSON serialization
	// issues. either solve the serialization issues or add options to dstest.CompareDatasets
	expect.Body = json.RawMessage{0x5b, 0x5d}
	expect.Stats.Qri = dataset.KindStats.String()

	if diff := dstest.CompareDatasets(expect, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
		dstest.UpdateGoldenFileIfEnvVarSet("testdata/expect/TestCreatePreview.turnstile.json", got)
	}

	nowTfRef := addNowTransformDataset(t, r)

	got, err = CreatePreview(ctx, r.Filesystem(), nowTfRef)
	if err != nil {
		t.Fatal(err)
	}

	expect = dstest.LoadGoldenFile(t, "testdata/expect/TestCreatePreview.transform.json")
	// TODO (b5) - required adjustments for accurate comparison due to JSON serialization
	// issues. either solve the serialization issues or add options to dstest.CompareDatasets
	expect.Body = json.RawMessage{0x5b, 0x5d}
	expect.Stats.Qri = dataset.KindStats.String()

	if diff := dstest.CompareDatasets(expect, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
		dstest.UpdateGoldenFileIfEnvVarSet("testdata/expect/TestCreatePreview.transform.json", got)
	}
}
