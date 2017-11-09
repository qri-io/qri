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
	// File:   "data/airport-codes.csv",
	// Readme: "readme.md",
	// Format: "text/csv",
}

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

var AirportCodesStructureAgebraic = &Structure{
	Format:       CsvDataFormat,
	FormatConfig: &CsvOptions{HeaderRow: true},
	Schema: &Schema{
		Fields: []*Field{
			&Field{
				Name: "col_0",
				Type: datatypes.String,
			},
			&Field{
				Name: "col_1",
				Type: datatypes.String,
			},
			&Field{
				Name: "col_2",
				Type: datatypes.String,
			},
			&Field{
				Name: "col_3",
				Type: datatypes.Float,
			},
			&Field{
				Name: "col_4",
				Type: datatypes.Float,
			},
			&Field{
				Name: "col_5",
				Type: datatypes.Integer,
			},
			&Field{
				Name: "col_6",
				Type: datatypes.String,
			},
			&Field{
				Name: "col_7",
				Type: datatypes.String,
			},
			&Field{
				Name: "col_8",
				Type: datatypes.String,
			},
			&Field{
				Name: "col_9",
				Type: datatypes.String,
			},
			&Field{
				Name: "col_10",
				Type: datatypes.String,
			},
			&Field{
				Name: "col_11",
				Type: datatypes.String,
			},
			&Field{
				Name: "col_12",
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
