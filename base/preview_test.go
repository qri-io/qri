package base

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
)

func TestCreatePreview(t *testing.T) {

	prevTs := dsfs.Timestamp
	dsfs.Timestamp = func() time.Time { return new(time.Time).In(time.UTC) }
	defer func() { dsfs.Timestamp = prevTs }()

	r := newTestRepo(t)
	turnstileRef := addTurnstileDataset(t, r)
	ctx := context.Background()

	got, err := CreatePreview(ctx, r, turnstileRef)
	if err != nil {
		t.Fatal(err)
	}

	ts, _ := time.Parse(time.RFC3339, "0001-01-01 00:00:00 +0000 UTC")

	expect := &dataset.Dataset{
		Qri:      "ds:0",
		Peername: "peer",
		Name:     "turnstile_daily_counts_2020",
		Path:     "/map/QmXrDtzEV7JXSZogXAqsmcj3497nZWRGMyJzEe1tmYV1cd",
		Commit: &dataset.Commit{
			Message:   "created dataset",
			Path:      "/map/QmbuPg3d9Nguze3uwEpGEtgYBZNoRVz4UQewwr3HcXTadu",
			Qri:       "cm:0",
			Signature: "Wj+Q8k+XVYSRc2kRNxfv1d6zJ/8Q+atH3bxOeQH/rYICovHI2D2OqUvI7Oaag4ka9f7vdjxnargmADDl3EuMUlx6vHsWbX64pQ2uMSOM7jya6T7o7URR9vyesko1rVTb8xVyDbZEcDY3+2hf2ZDgVCD5M0WSnqUTGRxT4O1kgOqIPn6GnzudYmNkV/jyi+U/uGzOUM6Au92gysc+vfIsXxgAYuJv3NVrHNYjI504L15nBAfnHPOYfWaUjBtJiyUN36auvP43+/aOxO/O8iK3TfPepO6ne+DMmSXymvqrbcBuuOQLUu8aOO7Z6YTDnUU/bl+9z349CxjhJ1nne8V3SA==",
			Timestamp: ts,
			Title:     "update data for week ending April 18, 2020",
		},
		Meta: &dataset.Meta{
			Description: "NYC Subway Turnstile Counts Data aggregated by day and station complex for the year 2020. Updated weekly.",
			Qri:         "md:0",
			Title:       "Turnstile Daily Counts 2020",
		},
		Readme: &dataset.Readme{
			Qri:        "rm:0",
			ScriptPath: "/map/QmQ93yKwktz778AiTjYPKwj1qqbvHDsWpYVff3Eicqn6Z5",
			ScriptBytes: []byte(`# nyc-transit-data/turnstile_daily_counts_2020

NYC Subway Turnstile Counts Data aggregated by day and station complex for the year 2020.  Updated weekly.

## Where the Data Came From

This aggregation was created from weekly raw turnstile counts published by the New York MTA at [http://web.mta.info/developers/turnstile.html](http://web.mta.info/developers/turnstile.html)

The raw data were imported into a postgresql database for processing, and aggregated to calendar days for each station complex.

The process is outlined in [this blog post](https://medium.com/qri-io/taming-the-mtas-unruly-turnstile-data-c945f5f96ba0), and the code for the data pipeline is [available on github](https://github.com/qri-io/data-stories-scripts/tree/master/nyc-turnstile-counts).

## Caveats

This aggregation is a best-effort to make a clean and usable dataset of station-level counts.  There were some assumptions and important decisions made to arrive at the finished product.

- The dataset excludes tur...`),
		},
		Transform: &dataset.Transform{
			Qri:        "tf:0",
			ScriptPath: "/map/QmXSce6KDQHLvi4AKDU8z7s4ouynKXKpD6TY7wJgF6reWM",
		},
		Structure: &dataset.Structure{
			Checksum: "QmTgK2uYPscacJ9KaBS8tryXRF5mvjuRbubF7h9bG2GgoN",
			Depth:    1,
			Format:   "json",
			Length:   2,
			Qri:      "st:0",
			Schema:   map[string]interface{}{"type": string("array")},
		},
		Body:     json.RawMessage(`[]`),
		BodyPath: "/map/QmTgK2uYPscacJ9KaBS8tryXRF5mvjuRbubF7h9bG2GgoN",
	}

	if diff := cmp.Diff(expect, got, cmpopts.IgnoreUnexported(dataset.Dataset{}, dataset.Meta{}, dataset.Readme{}, dataset.Transform{})); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}

	nowTfRef := addNowTransformDataset(t, r)

	got, err = CreatePreview(ctx, r, nowTfRef)
	if err != nil {
		t.Fatal(err)
	}

	expect = &dataset.Dataset{
		Qri:      "ds:0",
		Peername: "peer",
		Name:     "now_tf",
		Path:     "/map/QmShMXWEJ56XyiRUWk8q7Nvzphk7n7Jm7hg32Uf92S6yfq",
		Commit: &dataset.Commit{
			Message:   "created dataset",
			Path:      "/map/QmQ5gzPfZgw1PaSSS7hfzy3sp597pKfWqspsHwsdsR5DEG",
			Qri:       "cm:0",
			Signature: "Wj+Q8k+XVYSRc2kRNxfv1d6zJ/8Q+atH3bxOeQH/rYICovHI2D2OqUvI7Oaag4ka9f7vdjxnargmADDl3EuMUlx6vHsWbX64pQ2uMSOM7jya6T7o7URR9vyesko1rVTb8xVyDbZEcDY3+2hf2ZDgVCD5M0WSnqUTGRxT4O1kgOqIPn6GnzudYmNkV/jyi+U/uGzOUM6Au92gysc+vfIsXxgAYuJv3NVrHNYjI504L15nBAfnHPOYfWaUjBtJiyUN36auvP43+/aOxO/O8iK3TfPepO6ne+DMmSXymvqrbcBuuOQLUu8aOO7Z6YTDnUU/bl+9z349CxjhJ1nne8V3SA==",
			Timestamp: ts,
			Title:     "created dataset",
		},
		Meta: &dataset.Meta{
			Qri:   "md:0",
			Title: "example transform",
		},
		Readme: &dataset.Readme{
			Qri:         "rm:0",
			ScriptPath:  "/map/QmfTcGiaJqhddaEGebrfAWH25YZkpPL7MMTC9swzNnb1FS",
			ScriptBytes: []byte("# Oh hey there!\nI'm a readme! hello!\n"),
		},
		Transform: &dataset.Transform{
			Qri:        "tf:0",
			ScriptPath: "/map/QmXSce6KDQHLvi4AKDU8z7s4ouynKXKpD6TY7wJgF6reWM",
		},
		Structure: &dataset.Structure{
			Checksum: "QmTgK2uYPscacJ9KaBS8tryXRF5mvjuRbubF7h9bG2GgoN",
			Depth:    1,
			Format:   "json",
			Length:   2,
			Qri:      "st:0",
			Schema:   map[string]interface{}{"type": string("array")},
		},
		Body:     json.RawMessage(`[]`),
		BodyPath: "/map/QmTgK2uYPscacJ9KaBS8tryXRF5mvjuRbubF7h9bG2GgoN",
	}

	if diff := cmp.Diff(expect, got, cmpopts.IgnoreUnexported(dataset.Dataset{}, dataset.Meta{}, dataset.Readme{}, dataset.Transform{})); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
