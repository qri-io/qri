package dsfs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/cafs"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
)

var AirportCodes = &dataset.Dataset{
	Title:    "Airport Codes",
	Homepage: "http://www.ourairports.com/",
	License: &dataset.License{
		Type: "PDDL-1.0",
	},
	Citations: []*dataset.Citation{
		&dataset.Citation{
			Name: "Our Airports",
			Url:  "http://ourairports.com/data/",
		},
	},
	// File:   "data/airport-codes.csv",
	// Readme: "readme.md",
	// Format: "text/csv",
}

var AirportCodesStructure = &dataset.Structure{
	Format: dataset.CsvDataFormat,
	FormatConfig: &dataset.CsvOptions{
		HeaderRow: true,
	},
	Schema: &dataset.Schema{
		Fields: []*dataset.Field{
			&dataset.Field{
				Name: "ident",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "type",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "name",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "latitude_deg",
				Type: datatypes.Float,
			},
			&dataset.Field{
				Name: "longitude_deg",
				Type: datatypes.Float,
			},
			&dataset.Field{
				Name: "elevation_ft",
				Type: datatypes.Integer,
			},
			&dataset.Field{
				Name: "continent",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "iso_country",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "iso_region",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "municipality",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "gps_code",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "iata_code",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "local_code",
				Type: datatypes.String,
			},
		},
	},
}

var AirportCodesStructureAgebraic = &dataset.Structure{
	Format:       dataset.CsvDataFormat,
	FormatConfig: &dataset.CsvOptions{HeaderRow: true},
	Schema: &dataset.Schema{
		Fields: []*dataset.Field{
			&dataset.Field{
				Name: "col_0",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "col_1",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "col_2",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "col_3",
				Type: datatypes.Float,
			},
			&dataset.Field{
				Name: "col_4",
				Type: datatypes.Float,
			},
			&dataset.Field{
				Name: "col_5",
				Type: datatypes.Integer,
			},
			&dataset.Field{
				Name: "col_6",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "col_7",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "col_8",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "col_9",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "col_10",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "col_11",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "col_12",
				Type: datatypes.String,
			},
		},
	},
}

var ContinentCodes = &dataset.Dataset{
	Title:       "Continent Codes",
	Description: "list of continents with corresponding two letter codes",
	License: &dataset.License{
		Type: "odc-pddl",
		Url:  "http://opendatacommons.org/licenses/pddl/",
	},
	Keywords: []string{
		"Continents",
		"Two letter code",
		"Continent codes",
		"Continent code list",
	},
}

var ContinentCodesStructure = &dataset.Structure{
	Format: dataset.CsvDataFormat,
	Schema: &dataset.Schema{
		Fields: []*dataset.Field{
			&dataset.Field{
				Name: "Code",
				Type: datatypes.String,
			},
			&dataset.Field{
				Name: "Name",
				Type: datatypes.String,
			},
		},
	},
}

var Hours = &dataset.Dataset{
	Title: "hours",
	// Data:   datastore.NewKey("/ipfs/QmS1dVa1xemo7gQzJgjimj1WwnVBF3TwRTGsyKa1uEBWbJ"),
}

var HoursStructure = &dataset.Structure{
	Format: dataset.CsvDataFormat,
	Schema: &dataset.Schema{
		Fields: []*dataset.Field{
			&dataset.Field{Name: "field_1", Type: datatypes.Date},
			&dataset.Field{Name: "field_2", Type: datatypes.Float},
			&dataset.Field{Name: "field_3", Type: datatypes.String},
			&dataset.Field{Name: "field_4", Type: datatypes.String},
		},
	},
}

func makeFilestore() (map[string]datastore.Key, cafs.Filestore, error) {
	fs := memfs.NewMapstore()

	datasets := map[string]datastore.Key{
		"movies": datastore.NewKey(""),
		"cities": datastore.NewKey(""),
	}

	for k, _ := range datasets {
		rawdata, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s.csv", k))
		if err != nil {
			return datasets, nil, err
		}

		datakey, err := fs.Put(memfs.NewMemfileBytes(k, rawdata), true)
		if err != nil {
			return datasets, nil, err
		}

		dsdata, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s.json", k))
		if err != nil {
			return datasets, nil, err
		}

		ds := &dataset.Dataset{}
		if err := json.Unmarshal(dsdata, ds); err != nil {
			return datasets, nil, err
		}
		ds.Data = datakey
		dskey, err := SaveDataset(fs, ds, true)
		if err != nil {
			return datasets, nil, err
		}
		datasets[k] = dskey
	}

	return datasets, fs, nil
}
