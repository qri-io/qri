package dsref

import (
	"testing"
)

const TestPrefix = "prefix_"

func TestGenerateName(t *testing.T) {
	cases := []struct {
		description string
		input       string
		expect      string
	}{
		{
			"simple string",
			"data",
			"data",
		},
		{
			"title case",
			"Information",
			"information",
		},
		{
			"underscores stay the same",
			"body_input",
			"body_input",
		},
		{
			"spaces replaced",
			"sample population 2019",
			"sample_population_2019",
		},
		{
			"spaces are trimmed at start and end",
			" good name ",
			"good_name",
		},
		{
			"control characters are ignored",
			"some\tweird\rchar\ncodes\x07in\x7fname",
			"someweirdcharcodesinname",
		},
		{
			"punctuation followed by space makes one dash",
			"category: annual",
			"category-annual",
		},
		{
			"split words are separated by underscore",
			"final1997report",
			"final_1997_report",
		},
		{
			"many words separated by punctuation that will be cut",
			"en.wikipedia.org,List_of_highest-grossing_films,Highest-grossing_films",
			"en-wikipedia-org-list_of_highest-grossing",
		},
		{
			"lots of words that will cut off at word boundary",
			"Title, artist, date, and medium of every artwork in the MoMA collection",
			"title-artist-date-and_medium_of_every",
		},
		{
			"single long word that has to be truncated",
			"Pneumonoultramicroscopicsilicovolcanoconiosis",
			"pneumonoultramicroscopicsilicovolcanoconiosi",
		},
		{
			"single long word in Icelandic uses non-ascii characters",
			"Vaðlaheiðarvegavinnuverkfærageymsluskúrslyklakippuhringurinn",
			"valaheiarvegavinnuverkfrageymsluskurslyklaki",
		},
		{
			"start with number and contains spaces",
			"2018 winners",
			"prefix_2018_winners",
		},
		{
			"dashes stay",
			"2015-09-16..2016-09-30",
			"prefix_2015-09-16--2016-09-30",
		},
		{
			"unicode normalize to ascii",
			"pira\u00f1a_data",
			"pirana_data",
		},
		{
			"camel case with number",
			"EconIndicatorNominalGDP1997China",
			"econ_indicator_nominal_gdp_1997_china",
		},
		{
			"camel case with all caps word",
			"MonthlyHTTPTraffic",
			"monthly_http_traffic",
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			actual := GenerateName(c.input, TestPrefix)
			if actual != c.expect {
				t.Errorf("mismatch, expect: %s, got: %s", c.expect, actual)
			}
		})
	}
}

// Test the edge case of cutting off at a word boundary. The right-side always has a whole word,
// and does not ever have a space.
func TestGenerateNameCutoffWord(t *testing.T) {
	realLen := NameMaxLength
	defer func() {
		NameMaxLength = realLen
	}()
	NameMaxLength = 18

	cases := []struct {
		input       string
		expect      string
	}{
		{
			"the quick brown fox jumped",
			"the_quick_brown",
		},
		{
			"the quic brown fox jumped",
			"the_quic_brown_fox",
		},
		{
			"the qui brown fox jumped",
			"the_qui_brown_fox",
		},
		{
			"the qu brown fox jumped",
			"the_qu_brown_fox",
		},
		{
			"the q brown fox jumped",
			"the_q_brown_fox",
		},
		{
			"the  brown fox jumped",
			"the__brown_fox",
		},
		{
			"the brown fox jumped",
			"the_brown_fox",
		},
		{
			"the brow fox jumped",
			"the_brow_fox",
		},
		{
			"the bro fox jumped",
			"the_bro_fox_jumped",
		},
	}
	for i, c := range cases {
		actual := GenerateName(c.input, TestPrefix)
		if actual != c.expect {
			t.Errorf("case %d: mismatch, expect: %s, got: %s", i, c.expect, actual)
		}
	}
}
