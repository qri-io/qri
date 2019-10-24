package dsfs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qfs/cafs"
)

var AirportCodes = &dataset.Dataset{
	Meta: &dataset.Meta{
		Title:   "Airport Codes",
		HomeURL: "http://www.ourairports.com/",
		License: &dataset.License{
			Type: "PDDL-1.0",
		},
		Citations: []*dataset.Citation{
			{
				Name: "Our Airports",
				URL:  "http://ourairports.com/data/",
			},
		},
	},
	// File:   "data/airport-codes.csv",
	// Readme: "readme.md",
	// Format: "text/csv",
}

var AirportCodesCommit = &dataset.Commit{
	Qri:     dataset.KindCommit.String(),
	Message: "initial commit",
}

var AirportCodesStructure = &dataset.Structure{
	Format: "csv",
	FormatConfig: map[string]interface{}{
		"headerRow": true,
	},
	Schema: map[string]interface{}{
		"type": "array",
		"items": map[string]interface{}{
			"type": "array",
			"items": []interface{}{
				map[string]interface{}{"title": "ident", "type": "string"},
				map[string]interface{}{"title": "type", "type": "string"},
				map[string]interface{}{"title": "name", "type": "string"},
				map[string]interface{}{"title": "latitude_deg", "type": "number"},
				map[string]interface{}{"title": "longitude_deg", "type": "number"},
				map[string]interface{}{"title": "elevation_ft", "type": "integer"},
				map[string]interface{}{"title": "continent", "type": "string"},
				map[string]interface{}{"title": "iso_country", "type": "string"},
				map[string]interface{}{"title": "iso_region", "type": "string"},
				map[string]interface{}{"title": "municipality", "type": "string"},
				map[string]interface{}{"title": "gps_code", "type": "string"},
				map[string]interface{}{"title": "iata_code", "type": "string"},
				map[string]interface{}{"title": "local_code", "type": "string"},
			},
		},
	},
}

var AirportCodesStructureAgebraic = &dataset.Structure{
	Format:       "csv",
	FormatConfig: map[string]interface{}{"headerRow": true},
	Schema: map[string]interface{}{
		"type": "array",
		"items": map[string]interface{}{
			"type": "array",
			"items": []interface{}{
				map[string]interface{}{"title": "col_0", "type": "string"},
				map[string]interface{}{"title": "col_1", "type": "string"},
				map[string]interface{}{"title": "col_2", "type": "string"},
				map[string]interface{}{"title": "col_3", "type": "number"},
				map[string]interface{}{"title": "col_4", "type": "number"},
				map[string]interface{}{"title": "col_5", "type": "integer"},
				map[string]interface{}{"title": "col_6", "type": "string"},
				map[string]interface{}{"title": "col_7", "type": "string"},
				map[string]interface{}{"title": "col_8", "type": "string"},
				map[string]interface{}{"title": "col_9", "type": "string"},
				map[string]interface{}{"title": "col_10", "type": "string"},
				map[string]interface{}{"title": "col_11", "type": "string"},
				map[string]interface{}{"title": "col_12", "type": "string"},
			},
		},
	},
}

var ContinentCodes = &dataset.Dataset{
	Qri: dataset.KindDataset.String(),
	Meta: &dataset.Meta{
		Qri:         dataset.KindMeta.String(),
		Title:       "Continent Codes",
		Description: "list of continents with corresponding two letter codes",
		License: &dataset.License{
			Type: "odc-pddl",
			URL:  "http://opendatacommons.org/licenses/pddl/",
		},
		Keywords: []string{
			"Continents",
			"Two letter code",
			"Continent codes",
			"Continent code list",
		},
	},
}

var ContinentCodesStructure = &dataset.Structure{
	Format: "csv",
	Schema: map[string]interface{}{
		"type": "array",
		"items": map[string]interface{}{
			"type": "array",
			"items": []interface{}{
				map[string]interface{}{"title": "code", "type": "string"},
				map[string]interface{}{"title": "name", "type": "string"},
			},
		},
	},
}

var Hours = &dataset.Dataset{
	Meta: &dataset.Meta{
		Title: "hours",
	},
	// Body:   "/ipfs/QmS1dVa1xemo7gQzJgjimj1WwnVBF3TwRTGsyKa1uEBWbJ",
}

var HoursStructure = &dataset.Structure{
	Format: "csv",
	Schema: map[string]interface{}{
		"type": "array",
		"items": map[string]interface{}{
			"type": "array",
			"items": []interface{}{
				map[string]interface{}{"title": "field_1", "type": "string"},
				map[string]interface{}{"title": "field_2", "type": "number"},
				map[string]interface{}{"title": "field_3", "type": "string"},
				map[string]interface{}{"title": "field_4", "type": "string"},
			},
		},
	},
}

func makeFilestore() (map[string]string, cafs.Filestore, error) {
	ctx := context.Background()
	st := cafs.NewMapstore()

	datasets := map[string]string{
		"movies": "",
		"cities": "",
	}

	for k := range datasets {
		dsdata, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s/input.dataset.json", k))
		if err != nil {
			return datasets, nil, err
		}

		ds := &dataset.Dataset{}
		if err := json.Unmarshal(dsdata, ds); err != nil {
			return datasets, nil, err
		}

		dataPath := fmt.Sprintf("testdata/%s/body.%s", k, ds.Structure.Format)
		data, err := ioutil.ReadFile(dataPath)
		if err != nil {
			return datasets, nil, err
		}

		ds.SetBodyFile(qfs.NewMemfileBytes(filepath.Base(dataPath), data))

		dskey, err := WriteDataset(ctx, st, ds, true)
		if err != nil {
			return datasets, nil, fmt.Errorf("dataset: %s write error: %s", k, err.Error())
		}
		datasets[k] = dskey
	}

	return datasets, st, nil
}
