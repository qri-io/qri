package dsref

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
)

func TestConvertToVersionInfo(t *testing.T) {
	ds := &dataset.Dataset{
		Peername:  "a",
		ProfileID: "a's profile ID",
		Name:      "b",
		Path:      "c",
		Commit: &dataset.Commit{
			Timestamp: time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC),
			Title:     "d",
			Message:   "e",
		},
		Meta: &dataset.Meta{
			Title: "f",
			Theme: []string{"g"},
		},
		Structure: &dataset.Structure{
			Format:   "h",
			Length:   1,
			ErrCount: 2,
			Entries:  3,
		},
	}

	expect := VersionInfo{
		Username:   "a",
		ProfileID:  "a's profile ID",
		Name:       "b",
		Path:       "c",
		CommitTime: time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC),

		MetaTitle: "f",
		ThemeList: "g",

		BodyFormat: "h",
		BodySize:   1,
		NumErrors:  2,
		BodyRows:   3,
	}

	got := ConvertDatasetToVersionInfo(ds)

	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
