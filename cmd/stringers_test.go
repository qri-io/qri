package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/iso8601"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/repo"
)

func TestPeerStringer(t *testing.T) {
	setNoColor(false)
	cases := []struct {
		description string
		peer        *config.ProfilePod
		expect      string
	}{
		// Online & multiple NetworkAddrs
		{"Peer Stringer - Online & multiple addresses",
			&config.ProfilePod{
				Peername:     "cassie",
				Online:       true,
				ID:           "Qm...yay",
				NetworkAddrs: []string{"address_1", "address_2", "address_3"},
			}, "\u001b[32;1mcassie\u001b[0m | \u001b[33monline\u001b[0m\nProfile ID: Qm...yay\nAddresses:    address_1\n              address_2\n              address_3\n\n"},
		// online & no NetworkAddrs
		{"Peer Stringer - Online & no addresses",
			&config.ProfilePod{
				Peername: "justin",
				Online:   true,
				ID:       "Qm...woo",
			}, "\u001b[32;1mjustin\u001b[0m | \u001b[33monline\u001b[0m\nProfile ID: Qm...woo\n\n"},
		// Not online & one NetworkAddrs
		{"Peer Stringer - Not Online & one address",
			&config.ProfilePod{
				Peername:     "brandon",
				Online:       true,
				ID:           "Qm...hi",
				NetworkAddrs: []string{"address_1"},
			}, "\u001b[32;1mbrandon\u001b[0m | \u001b[33monline\u001b[0m\nProfile ID: Qm...hi\nAddress:    address_1\n\n"},
		// Not Online
		{"Peer Stringer - Not Online",
			&config.ProfilePod{
				Peername: "ricky",
				Online:   false,
				ID:       "Qm...wee",
			}, "\u001b[32;1mricky\u001b[0m\nProfile ID: Qm...wee\n\n"},
	}
	for _, c := range cases {
		peerStr := peerStringer(*c.peer).String()

		if c.expect != peerStr {
			t.Errorf("case '%s', expected: '%s', got'%s'", c.description, c.expect, peerStr)
		}
	}

}

func TestRefStringer(t *testing.T) {
	setNoColor(false)
	cases := []struct {
		description string
		ref         *repo.DatasetRef
		expect      string
	}{
		{"RefStringer - all fields, singular",
			&repo.DatasetRef{
				Name:     "ds_name",
				Peername: "peer",
				Path:     "/network/hash",
				Dataset: &dataset.Dataset{
					Structure: &dataset.Structure{
						Length:   1,
						Entries:  1,
						ErrCount: 1,
					},
					NumVersions: 1,
					Meta: &dataset.Meta{
						Title: "Dataset Title",
					},
				},
			}, "\u001b[32;1mpeer/ds_name\u001b[0m\nDataset Title\n\u001b[2m/network/hash\u001b[0m\n1 B, 1 entry, 1 error, 1 version\n\n",
		},
		{"RefStringer - all fields, plural",
			&repo.DatasetRef{
				Name:     "ds_name",
				Peername: "peer",
				Path:     "/network/hash",
				Dataset: &dataset.Dataset{
					Structure: &dataset.Structure{
						Length:   10,
						Entries:  10,
						ErrCount: 10,
					},
					NumVersions: 10,
					Meta: &dataset.Meta{
						Title: "Dataset Title",
					},
				},
			}, "\u001b[32;1mpeer/ds_name\u001b[0m\nDataset Title\n\u001b[2m/network/hash\u001b[0m\n10 B, 10 entries, 10 errors, 10 versions\n\n",
		},
		{"RefStringer - only peername & name",
			&repo.DatasetRef{
				Peername: "peer",
				Name:     "ds_name",
			}, "\u001b[32;1mpeer/ds_name\u001b[0m\n\n",
		},
	}
	for _, c := range cases {
		refStr := refStringer(*c.ref).String()
		if c.expect != refStr {
			t.Errorf("case '%s', expected: '%s', got'%s'", c.description, c.expect, refStr)
		}
	}
}

func TestLogStringer(t *testing.T) {
	setNoColor(false)
	time := time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC)
	cases := []struct {
		description string
		log         *repo.DatasetRef
		expect      string
	}{
		{"LogStringer - all fields",
			&repo.DatasetRef{
				Peername: "peer",
				Path:     "/network/hash",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: time,
						Title:     "commit title",
						Message:   "commit message",
					},
				},
			}, "\u001b[32mpath:   /network/hash\u001b[0m\nAuthor: peer\nDate:   Jan  1 01:01:01\n\n    commit title\n    commit message\n\n",
		},
		{"LogStringer - no message",
			&repo.DatasetRef{
				Peername: "peer",
				Path:     "/network/hash",
				Dataset: &dataset.Dataset{
					Commit: &dataset.Commit{
						Timestamp: time,
						Title:     "commit title",
					},
				},
			}, "\u001b[32mpath:   /network/hash\u001b[0m\nAuthor: peer\nDate:   Jan  1 01:01:01\n\n    commit title\n\n",
		},
	}
	for _, c := range cases {
		logStr := logStringer(*c.log).String()
		if c.expect != logStr {
			t.Errorf("case '%s', expected: '%s', got'%s'", c.description, c.expect, logStr)
		}
	}
}

func TestJobStringer(t *testing.T) {
	// NB: JobStringer is tough to write tests for at the moment
	// thanks to printing in local timezones
	// TODO (b5) - look into setting local timezone for this test

	setNoColor(false)
	time := time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC)
	p, err := iso8601.ParseRepeatingInterval("R/P1D")
	if err != nil {
		t.Error(err)
	}

	cases := []struct {
		description string
		job         *lib.Job
		contains      string
	}{
		{"JobStringer - all fields",
			&lib.Job{
				Name:         "Job",
				Type:         "dataset",
				Periodicity:  p,
				LastRunStart: time,
			}, "\u001b[1mJob\u001b[0m\n",
		},
	}
	for _, c := range cases {
		jobStr := jobStringer(*c.job).String()
		if !strings.Contains(jobStr, c.contains) {
			t.Errorf("case '%s', expected '%s' to contain string: '%s'", c.description, jobStr, c.contains)
		}
	}
}
