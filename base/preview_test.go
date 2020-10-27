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

	got, err := CreatePreview(ctx, r.Filesystem(), turnstileRef)
	if err != nil {
		t.Fatal(err)
	}

	ts, _ := time.Parse(time.RFC3339, "0001-01-01 00:00:00 +0000 UTC")

	expect := &dataset.Dataset{
		Qri:      "ds:0",
		Peername: "peer",
		Name:     "turnstile_daily_counts_2020",
		Path:     "/mem/QmSsaaJ9i7rKvYQ7vgLnwvJUfbPLtinwUvjjg5bXyMKmA4",
		Commit: &dataset.Commit{
			Message:   "created dataset",
			Path:      "/mem/QmPtRiULjivVKtT5cxhQUqtodZA4xPBbCWfW79KzBRQeoZ",
			Qri:       "cm:0",
			Signature: "JVDjt9Dd3y2wjaiYbaKDdSEWuGwy+8O2OWfAHhMJ4gThJLD+DTFplw5cqUgUg7mi0QJjAwvR4AYpIWfyjHJQdRO5t9PQjuEQM9LSI1zrmL7EHrb8isqVyAH6n4/sRSVR23/9ilSHGGmjeqmRopkbbGsI8KhUlNo32RjIhF1NhMJ/d8mVy3lRhg/U64/+Vav6DWCBWKZUxa45bNKCmZgSXF8bzedFZ838363OIm0o7i82S56RIT8kDzhl3ubMmloQBBGxpl+Ylcs1KqE8UXjVQhG+tHvbUb3AXIXXx+Nek5bAx9N4/j+3Rmnwm56h/y/oBrL8e6GNH/iq7fwMo+gMfA==",
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
			ScriptPath: "/mem/QmQ93yKwktz778AiTjYPKwj1qqbvHDsWpYVff3Eicqn6Z5",
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
			ScriptPath: "/mem/QmXSce6KDQHLvi4AKDU8z7s4ouynKXKpD6TY7wJgF6reWM",
		},
		Structure: &dataset.Structure{
			Depth:  1,
			Format: "json",
			Length: 2,
			Qri:    "st:0",
			Schema: map[string]interface{}{"type": string("array")},
		},
		Body:     json.RawMessage(`[]`),
		BodyPath: "/mem/QmTgK2uYPscacJ9KaBS8tryXRF5mvjuRbubF7h9bG2GgoN",
	}

	if diff := cmp.Diff(expect, got, cmpopts.IgnoreUnexported(dataset.Dataset{}, dataset.Meta{}, dataset.Readme{}, dataset.Transform{})); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}

	nowTfRef := addNowTransformDataset(t, r)

	got, err = CreatePreview(ctx, r.Filesystem(), nowTfRef)
	if err != nil {
		t.Fatal(err)
	}

	expect = &dataset.Dataset{
		Qri:      "ds:0",
		Peername: "peer",
		Name:     "now_tf",
		Path:     "/mem/QmUycGKj2wmbk2R7n3mGwFuV8FTeu8AL4bZbvz4RXjqeNL",
		Commit: &dataset.Commit{
			Message:   "created dataset",
			Path:      "/mem/QmQnGzn3bzB3TqKtpskFqBDtpchsPt946JAP6HXNtXCWyk",
			Qri:       "cm:0",
			Signature: "JVDjt9Dd3y2wjaiYbaKDdSEWuGwy+8O2OWfAHhMJ4gThJLD+DTFplw5cqUgUg7mi0QJjAwvR4AYpIWfyjHJQdRO5t9PQjuEQM9LSI1zrmL7EHrb8isqVyAH6n4/sRSVR23/9ilSHGGmjeqmRopkbbGsI8KhUlNo32RjIhF1NhMJ/d8mVy3lRhg/U64/+Vav6DWCBWKZUxa45bNKCmZgSXF8bzedFZ838363OIm0o7i82S56RIT8kDzhl3ubMmloQBBGxpl+Ylcs1KqE8UXjVQhG+tHvbUb3AXIXXx+Nek5bAx9N4/j+3Rmnwm56h/y/oBrL8e6GNH/iq7fwMo+gMfA==",
			Timestamp: ts,
			Title:     "created dataset",
		},
		Meta: &dataset.Meta{
			Qri:   "md:0",
			Title: "example transform",
		},
		Readme: &dataset.Readme{
			Qri:         "rm:0",
			ScriptPath:  "/mem/QmfTcGiaJqhddaEGebrfAWH25YZkpPL7MMTC9swzNnb1FS",
			ScriptBytes: []byte("# Oh hey there!\nI'm a readme! hello!\n"),
		},
		Transform: &dataset.Transform{
			Qri:        "tf:0",
			ScriptPath: "/mem/QmXSce6KDQHLvi4AKDU8z7s4ouynKXKpD6TY7wJgF6reWM",
		},
		Structure: &dataset.Structure{
			Depth:  1,
			Format: "json",
			Length: 2,
			Qri:    "st:0",
			Schema: map[string]interface{}{"type": string("array")},
		},
		Body:     json.RawMessage(`[]`),
		BodyPath: "/mem/QmTgK2uYPscacJ9KaBS8tryXRF5mvjuRbubF7h9bG2GgoN",
	}

	if diff := cmp.Diff(expect, got, cmpopts.IgnoreUnexported(dataset.Dataset{}, dataset.Meta{}, dataset.Readme{}, dataset.Transform{})); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
