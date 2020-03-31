package archive

import (
	"testing"
	"time"

	"github.com/qri-io/dataset"
)

func TestGenerateFilename(t *testing.T) {
	// no commit
	// no structure & no format
	// no format & yes structure
	// timestamp and format!
	loc := time.FixedZone("UTC-8", -8*60*60)
	timeStamp := time.Date(2009, time.November, 10, 23, 0, 0, 0, loc)
	cases := []struct {
		description string
		ds          *dataset.Dataset
		format      string
		expected    string
		err         string
	}{
		{
			"no format & no structure",
			&dataset.Dataset{}, "", "", "no format specified and no format present in the dataset Structure",
		},
		{
			"no format & no Structure.Format",
			&dataset.Dataset{
				Structure: &dataset.Structure{
					Format: "",
				},
			}, "", "", "no format specified and no format present in the dataset Structure",
		},
		{
			"no format specified & Structure.Format exists",
			&dataset.Dataset{
				Commit: &dataset.Commit{
					Timestamp: timeStamp,
				},
				Structure: &dataset.Structure{
					Format: "json",
				},
				Peername: "cassie",
				Name:     "fun_dataset",
			}, "", "cassie-fun_dataset_-_2009-11-10-23-00-00.json", "",
		},
		{
			"no format specified & Structure.Format exists",
			&dataset.Dataset{
				Commit: &dataset.Commit{
					Timestamp: timeStamp,
				},
				Structure: &dataset.Structure{
					Format: "json",
				},
				Peername: "brandon",
				Name:     "awesome_dataset",
			}, "", "brandon-awesome_dataset_-_2009-11-10-23-00-00.json", "",
		},
		{
			"format: json",
			&dataset.Dataset{
				Commit: &dataset.Commit{
					Timestamp: timeStamp,
				},
				Peername: "ricky",
				Name:     "rad_dataset",
			}, "json", "ricky-rad_dataset_-_2009-11-10-23-00-00.json", "",
		},
		{
			"format: csv",
			&dataset.Dataset{
				Commit: &dataset.Commit{
					Timestamp: timeStamp,
				},
				Peername: "justin",
				Name:     "cool_dataset",
			}, "csv", "justin-cool_dataset_-_2009-11-10-23-00-00.csv", "",
		},
		{
			"no timestamp",
			&dataset.Dataset{
				Peername: "no",
				Name:     "time",
			}, "csv", "no-time_-_0001-01-01-00-00-00.csv", "",
		},
	}
	for _, c := range cases {
		got, err := GenerateFilename(c.ds, c.format)
		if !(err == nil && c.err == "" || err != nil && err.Error() == c.err) {
			t.Errorf("case '%s' error mismatched: expected: '%s', got: '%s'", c.description, c.err, err)
		}
		if got != c.expected {
			t.Errorf("case '%s' filename mismatched: expected: '%s', got: '%s'", c.description, c.expected, got)
		}
	}
}
