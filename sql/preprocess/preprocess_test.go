package preprocess

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPrepropcess(t *testing.T) {
	good := []struct {
		pre, post string
		mapping   map[string]string
	}{
		{
			"select * from b5/country_codes@/ipfs/QmFoo t1",
			"select * from b5_country_codes_at__ipfs_QmFoo t1",
			map[string]string{
				"b5_country_codes_at__ipfs_QmFoo": "b5/country_codes@/ipfs/QmFoo",
			},
		},
		{
			"select foo from b5/country_codes",
			"select ds.foo from b5_country_codes ds",
			map[string]string{
				"b5_country_codes": "b5/country_codes",
			},
		},
		{
			"select * from b5/country_codes, b5/world_bank_population",
			"select * from b5_country_codes ds, b5_world_bank_population ds2",
			map[string]string{
				"b5_country_codes":         "b5/country_codes",
				"b5_world_bank_population": "b5/world_bank_population",
			},
		},
		{
			"select * from b5/covid_19_confirmed c limit 1",
			"select * from b5_covid_19_confirmed c limit 1",
			map[string]string{
				"b5_covid_19_confirmed": "b5/covid_19_confirmed",
			},
		},
		{
			"select 'from' from b5/country_codes@/ipfs/QmFoo as t1",
			"select t1.from from b5_country_codes_at__ipfs_QmFoo as t1",
			map[string]string{
				"b5_country_codes_at__ipfs_QmFoo": "b5/country_codes@/ipfs/QmFoo",
			},
		},
		{
			`SELECT c.official_name_en FROM b5/country_codes as c GROUP BY c.continent`,
			`SELECT c.official_name_en FROM b5_country_codes as c GROUP BY c.continent`,
			map[string]string{
				"b5_country_codes": "b5/country_codes",
			},
		},
		{
			"select * from peer/dataset_a as a, peer/dataset_b b order by b.name",
			"select * from peer_dataset_a as a, peer_dataset_b b order by b.name",
			map[string]string{
				"peer_dataset_a": "peer/dataset_a",
				"peer_dataset_b": "peer/dataset_b",
			},
		},
		{
			"SELECT (SELECT 1 FROM foo/dataset t1) as t2",
			"SELECT (SELECT 1 FROM foo_dataset t1) as t2",
			map[string]string{
				"foo_dataset": "foo/dataset",
			},
		},

		{
			"SELECT * FROM nyc-transit-data/turnstile_daily_counts_2019 t LIMIT 10",
			"SELECT * FROM nyc_transit_data_turnstile_daily_counts_2019 t LIMIT 10",
			map[string]string{
				"nyc_transit_data_turnstile_daily_counts_2019": "nyc-transit-data/turnstile_daily_counts_2019",
			},
		},

		{
			"SELECT (SELECT 1)",
			"SELECT (SELECT 1)",
			map[string]string{},
		},
		{
			"SELECT 1 FROM (SELECT 2 FROM (SELECT 3 FROM foo/dataset a) t1, peer/dataset b) AS c, b5/world_bank_population d",
			"SELECT 1 FROM (SELECT 2 FROM (SELECT 3 FROM foo_dataset a) t1, peer_dataset b) AS c, b5_world_bank_population d",
			map[string]string{
				"foo_dataset":              "foo/dataset",
				"peer_dataset":             "peer/dataset",
				"b5_world_bank_population": "b5/world_bank_population",
			},
		},
		{
			"SELECT 1 FROM (SELECT 2 FROM (SELECT 3 FROM foo/dataset), peer/dataset b) AS c, b5/world_bank_population",
			"SELECT 1 FROM (SELECT 2 FROM (SELECT 3 FROM foo_dataset ds) ds2, peer_dataset b) AS c, b5_world_bank_population ds3",
			map[string]string{
				"foo_dataset":              "foo/dataset",
				"peer_dataset":             "peer/dataset",
				"b5_world_bank_population": "b5/world_bank_population",
			},
		},
		{
			`SELECT 'c.province/state', 'c.country/region', 'c.lat', 'c.long', 'c.3/12/20' FROM b5/covid_19_confirmed as c`,
			`SELECT 'c.province/state', 'c.country/region', 'c.lat', 'c.long', 'c.3/12/20' FROM b5_covid_19_confirmed as c`,
			map[string]string{
				"b5_covid_19_confirmed": "b5/covid_19_confirmed",
			},
		},
		{
			`SELECT 'c.confirmed' as 'type', 'c.province/state', 'c.country/region', 'c.lat', 'c.long', 'c.3/12/20' FROM b5/covid_19_confirmed as c
			 UNION
			 SELECT 'r.recovered' as 'type', 'r.province/state', 'r.country/region', 'r.lat', 'r.long', 'c.3/12/20' FROM b5/covid_19_recovered as r`,
			`SELECT 'c.confirmed' as 'type', 'c.province/state', 'c.country/region', 'c.lat', 'c.long', 'c.3/12/20' FROM b5_covid_19_confirmed as c
			 UNION
			 SELECT 'r.recovered' as 'type', 'r.province/state', 'r.country/region', 'r.lat', 'r.long', 'c.3/12/20' FROM b5_covid_19_recovered as r`,
			map[string]string{
				"b5_covid_19_confirmed": "b5/covid_19_confirmed",
				"b5_covid_19_recovered": "b5/covid_19_recovered",
			},
		},
	}

	for _, c := range good {
		t.Run(c.pre, func(t *testing.T) {
			post, mapping, err := Query(c.pre)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(c.post, post); diff != "" {
				t.Errorf("altered query mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(c.mapping, mapping); diff != "" {
				t.Errorf("altered query mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPreprocessErrors(t *testing.T) {
	bad := []struct {
		input, err string
	}{
		{
			"SELECT * FROM foo b, bar b",
			`duplicate reference alias 'b'`,
		},
		{
			"SELECT * FROM foo b,,",
			"encountered ',' before table name",
		},
		// {
		// 	"SELECT * FROM foo b, bar b",
		// 	"duplicate reference alias 'b'",
		// },
	}
	for _, c := range bad {
		t.Run(c.input, func(t *testing.T) {
			_, _, err := Query(c.input)
			if err == nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(c.err, err.Error()); diff != "" {
				t.Errorf("expected error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
