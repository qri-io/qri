package cmd

import (
	"bytes"
	"encoding/csv"

	"github.com/qri-io/jsontable"
)

func readJsonData(schema *jsontable.Schema, data []byte) ([][]interface{}, error) {
	return nil, nil
}

func readCsvData(schema *jsontable.Schema, data []byte) ([][]interface{}, error) {
	r := csv.NewReader(bytes.NewReader(data))
	return nil, nil
}
