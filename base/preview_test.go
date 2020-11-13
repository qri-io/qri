package base

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

	ts, _ := time.Parse(time.RFC3339, "0001-01-01 00:00:00 +0000 UTC")

	expect := &dataset.Dataset{
		Qri:      "ds:0",
		Peername: "peer",
		Name:     "turnstile_daily_counts_2020",
		Path:     "/mem/QmacSZkNd6hL62Xhc9Uo8xe7VBXzTAwyakhnPL4mzwpngt",
		Commit: &dataset.Commit{
			Message:   "created dataset",
			Path:      "/mem/QmZvtU6nR3ncHnocgPFFBykP8M5dLXcaV4993etbBzUvRM",
			Qri:       "cm:0",
			Signature: "epknQl+k0RlM+w3g8QueNmXoDM3Z2j9O4CV8/d4p5p+nIRqKFTRsW5K/FNvZde4Qv5fl3ZfZ/23GyVPBKbmycdEGse02bLS4fTJHbebNPycaBNwHQ4M4FQwX0jYMezAjz+9ODSXpd28P7bkuPA0QBMCBcuRvZH8AVOuv4IJVdaUQU4uQeAnTf9EnSyORPM49XRsY0U5/5GP0l1hpVXEUVgOgOSsBAE9osMHY75P6gBL2FqwJgWoJw7VIph2zUCUNlV6JlIkoGD59qwO4dhzD6ITtYGX6xwLr7o0T8w8p67d9PAbHx/czHa/tFV6bnRxz0ViCBkAWzhimgBwqI16r4g==",
			Timestamp: ts,
			Title:     "update data for week ending April 18, 2020",
		},
		Meta: &dataset.Meta{
			Description: "NYC Subway Turnstile Counts Data aggregated by day and station complex for the year 2020. Updated weekly.",
			Qri:         "md:0",
			Title:       "Turnstile Daily Counts 2020",
			Path:        "/mem/QmZ3ruG85fa1wHmZnZ4QcwA6y52YmSeFDx3iuW1R5E7xRd",
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
			Path:   "/mem/QmbDN5ERopH2VBwuBLGo8JY1kYr8ZiG3o4snxuNR6XwpmS",
		},
		Body:     json.RawMessage(`[]`),
		BodyPath: "/mem/QmTgK2uYPscacJ9KaBS8tryXRF5mvjuRbubF7h9bG2GgoN",
		Stats:    &dataset.Stats{Path: "/mem/QmdzwcjZDAwPQJHikR9jnPg7x1jrKGseW2EZvd4s3eZxfG", Qri: "sa:0"},
	}

	if diff := dstest.CompareDatasets(expect, got); diff != "" {
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
		Path:     "/mem/QmNPFuufGRhkgrLoGEN4yYX4gS3s7TrpdBuRUqMaRXxyva",
		Commit: &dataset.Commit{
			Message:   "created dataset",
			Path:      "/mem/QmVSqa5by5nfirScnJnL2icZUvhFLsaVE3RH6SB5uvZnpA",
			Qri:       "cm:0",
			Signature: "snCNPPyyVoeIiVyezEpPWBURYfpE2ZL6XTrve+rxjNeEZSDvSUTGbDE0l9tLvFF7+Wu/nqVIKvwCnnyZ51/alOhoSMT4EcbcaXRnVvCJQeqqj7NRZNQ9vIhVbWpxN5b9MWrhS+LydnE3mOsnI/jtFhx3N5Zs6tGOiiRVtoAe5IKYhnEk8NBUGpbYLQuw74wQlT60OnDroFxhabGMFF4O+9bp/RKKqk/wQEPWaw511PL7/CY1xvT+y1YZ7sPwGXPFkWKX49qd48YXXeKVaA8f89UCBMpQ6egcogau12DbiYHp2cPXlMUSsbmG0QgJJve/sCSLJq/imfaFXW7mOptaig==",
			Timestamp: ts,
			Title:     "created dataset",
		},
		Meta: &dataset.Meta{
			Qri:   "md:0",
			Title: "example transform",
			Path:  "/mem/Qmb1kx8uNva2wxgDPKjuSxJz9g3W2RdgABWrrxCEu91ZT4",
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
			Path:   "/mem/QmbDN5ERopH2VBwuBLGo8JY1kYr8ZiG3o4snxuNR6XwpmS",
			Schema: map[string]interface{}{"type": string("array")},
		},
		Body:     json.RawMessage(`[]`),
		BodyPath: "/mem/QmTgK2uYPscacJ9KaBS8tryXRF5mvjuRbubF7h9bG2GgoN",
		Stats:    &dataset.Stats{Path: "/mem/QmdzwcjZDAwPQJHikR9jnPg7x1jrKGseW2EZvd4s3eZxfG", Qri: "sa:0"},
	}

	if diff := dstest.CompareDatasets(expect, got); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
