package wftest

import (
	"testing"
)

func TestLoadDefaultTestCases(t *testing.T) {
	_, err := LoadDefaultTestCases()
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadTestCases(t *testing.T) {
	_, err := LoadTestCases("./testdata")
	if err != nil {
		t.Fatal(err)
	}
}
