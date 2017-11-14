package dataset

import (
	"github.com/qri-io/dataset/datatypes"
)

var AirportCodes = &Dataset{
	Title:    "Airport Codes",
	Homepage: "http://www.ourairports.com/",
	License: &License{
		Type: "PDDL-1.0",
	},
	Citations: []*Citation{
		&Citation{
			Name: "Our Airports",
			Url:  "http://ourairports.com/data/",
		},
	},
	Commit:    &CommitMsg{Title: "initial commit"},
	Structure: AirportCodesStructure,
	// File:   "data/airport-codes.csv",
	// Readme: "readme.md",
	// Format: "text/csv",
}

const AirportCodesJSON = `{"citations":[{"name":"Our Airports","url":"http://ourairports.com/data/"}],"commit":{"title":"initial commit"},"data":"","homepage":"http://www.ourairports.com/","length":0,"license":"PDDL-1.0","structure":{"format":"csv","formatConfig":{"headerRow":true},"schema":{"fields":[{"name":"ident","type":"string"},{"name":"type","type":"string"},{"name":"name","type":"string"},{"name":"latitude_deg","type":"float"},{"name":"longitude_deg","type":"float"},{"name":"elevation_ft","type":"integer"},{"name":"continent","type":"string"},{"name":"iso_country","type":"string"},{"name":"iso_region","type":"string"},{"name":"municipality","type":"string"},{"name":"gps_code","type":"string"},{"name":"iata_code","type":"string"},{"name":"local_code","type":"string"}]}},"timestamp":"0001-01-01T00:00:00Z","title":"Airport Codes"}`

var AirportCodesStructure = &Structure{
	Format: CsvDataFormat,
	FormatConfig: &CsvOptions{
		HeaderRow: true,
	},
	Schema: &Schema{
		Fields: []*Field{
			&Field{
				Name: "ident",
				Type: datatypes.String,
			},
			&Field{
				Name: "type",
				Type: datatypes.String,
			},
			&Field{
				Name: "name",
				Type: datatypes.String,
			},
			&Field{
				Name: "latitude_deg",
				Type: datatypes.Float,
			},
			&Field{
				Name: "longitude_deg",
				Type: datatypes.Float,
			},
			&Field{
				Name: "elevation_ft",
				Type: datatypes.Integer,
			},
			&Field{
				Name: "continent",
				Type: datatypes.String,
			},
			&Field{
				Name: "iso_country",
				Type: datatypes.String,
			},
			&Field{
				Name: "iso_region",
				Type: datatypes.String,
			},
			&Field{
				Name: "municipality",
				Type: datatypes.String,
			},
			&Field{
				Name: "gps_code",
				Type: datatypes.String,
			},
			&Field{
				Name: "iata_code",
				Type: datatypes.String,
			},
			&Field{
				Name: "local_code",
				Type: datatypes.String,
			},
		},
	},
}

var AirportCodesStructureAbstract = &Structure{
	Format:       CsvDataFormat,
	FormatConfig: &CsvOptions{HeaderRow: true},
	Schema: &Schema{
		Fields: []*Field{
			&Field{
				Name: "a",
				Type: datatypes.String,
			},
			&Field{
				Name: "b",
				Type: datatypes.String,
			},
			&Field{
				Name: "c",
				Type: datatypes.String,
			},
			&Field{
				Name: "d",
				Type: datatypes.Float,
			},
			&Field{
				Name: "e",
				Type: datatypes.Float,
			},
			&Field{
				Name: "f",
				Type: datatypes.Integer,
			},
			&Field{
				Name: "g",
				Type: datatypes.String,
			},
			&Field{
				Name: "h",
				Type: datatypes.String,
			},
			&Field{
				Name: "i",
				Type: datatypes.String,
			},
			&Field{
				Name: "j",
				Type: datatypes.String,
			},
			&Field{
				Name: "k",
				Type: datatypes.String,
			},
			&Field{
				Name: "l",
				Type: datatypes.String,
			},
			&Field{
				Name: "m",
				Type: datatypes.String,
			},
		},
	},
}

var ContinentCodes = &Dataset{
	Title:       "Continent Codes",
	Description: "list of continents with corresponding two letter codes",
	License: &License{
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

var ContinentCodesStructure = &Structure{
	Format: CsvDataFormat,
	Schema: &Schema{
		Fields: []*Field{
			&Field{
				Name: "Code",
				Type: datatypes.String,
			},
			&Field{
				Name: "Name",
				Type: datatypes.String,
			},
		},
	},
}

var Hours = &Dataset{
	Title: "hours",
	// Data:   datastore.NewKey("/ipfs/QmS1dVa1xemo7gQzJgjimj1WwnVBF3TwRTGsyKa1uEBWbJ"),
}

var HoursStructure = &Structure{
	Format: CsvDataFormat,
	Schema: &Schema{
		Fields: []*Field{
			&Field{Name: "field_1", Type: datatypes.Date},
			&Field{Name: "field_2", Type: datatypes.Float},
			&Field{Name: "field_3", Type: datatypes.String},
			&Field{Name: "field_4", Type: datatypes.String},
		},
	},
}
