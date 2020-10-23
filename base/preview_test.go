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
		Path:     "/map/QmU9ZjZPMjdWtteTc25gF4rp2QmeEzx2nYYh671JtTCJ49",
		Commit: &dataset.Commit{
			Message:   "created dataset",
			Path:      "/map/QmW9k7TDg1Drieo1in6QWNjYNSuy8in2AjXDs3ai5faksp",
			Qri:       "cm:0",
			Signature: "r6UOq99ki0SW3pnX/RrIGO4chf0qm5ABjgiT9vZhz+wEJ864v2/XwST5EOtPbaECyeChq6a9tmKkVIBgEFNG3yH89AtX/zNMY63yrH8WOMy8rH1yWgPcQhmwghHqvwApPlHz6gHKDM2HkNjgS4VFMk2CJDsgJoU2VpEI2RKe5gQ+WHW/tfqWEUgHu1lKR7NmoodnjcF8fSeNvpYhL5aBjE+Z1y7xQ0/CWMgis6MmObqayZVpS8cwy79dCMEp1U84lMlzAZA8XX4SvF3uZn6JEhzpcq+Vq0h9CVG74dfQbP2UkVjtDaFqVS46uvQhHE+UD+7CPGa8t2b1eQsFkmEaDQ==",
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
			Depth:  1,
			Format: "json",
			Length: 2,
			Qri:    "st:0",
			Schema: map[string]interface{}{"type": string("array")},
		},
		Body:     json.RawMessage(`[]`),
		BodyPath: "/map/QmTgK2uYPscacJ9KaBS8tryXRF5mvjuRbubF7h9bG2GgoN",
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
		Path:     "/map/QmeUEtEa5WaWzbtmyqQXiRxF3eGMbkL7bzTML2GDwhi1Rb",
		Commit: &dataset.Commit{
			Message:   "created dataset",
			Path:      "/map/QmRK3ZhbkGph2NncPQyXnvmpwbs3sTp7mUYcs7f8HGtcAj",
			Qri:       "cm:0",
			Signature: "bGi4rYdk7tEiH+MdEq4DOc5DhUTRXVnE5w6NWuWZrmMj+0HcnslaheK/zmhylnsq264ldWDefAD2T7OjgvOIQTadEg64FMYV+YgOGY/IFZV0OTTZ/uXCO/QmOCSDwngKaLLcQXftYuVL0DUaUQul+8jg0BF1v2GnoAD13adOvaaJsymzHt+YpXUMUFbOa6HYJ01B2B2x+QmRKL7jx3MM4HJghlptrR0N8/NX8RmIkqvDHyuS+yiBzfwYAYbIYwh53hfFBDuzaVCYbszqVLUDAqTSG0LaxzE61VrkTilyKUqNfGk9a/Y61K2q96io0Mx6hoLRn5r/Lf94iFeRVyvAbw==",
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
			Depth:  1,
			Format: "json",
			Length: 2,
			Qri:    "st:0",
			Schema: map[string]interface{}{"type": string("array")},
		},
		Body:     json.RawMessage(`[]`),
		BodyPath: "/map/QmTgK2uYPscacJ9KaBS8tryXRF5mvjuRbubF7h9bG2GgoN",
	}

	if diff := cmp.Diff(expect, got, cmpopts.IgnoreUnexported(dataset.Dataset{}, dataset.Meta{}, dataset.Readme{}, dataset.Transform{})); diff != "" {
		t.Errorf("result mismatch. (-want +got):\n%s", diff)
	}
}
