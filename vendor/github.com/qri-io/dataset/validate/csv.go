package validate

import (
	"encoding/csv"
	"fmt"
	"io"
)

// CheckCsvRowLengths ensures that csv input has
// the same number of columns in every row and otherwise
// returns an error
func CheckCsvRowLengths(r io.Reader) error {
	csvReader := csv.NewReader(r)
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true
	//csvReader.LazyQuotes = true
	firstRow, err := csvReader.Read()
	rowLen := len(firstRow)
	if err != nil {
		return fmt.Errorf("error reading first row of csv: %s", err.Error())
	}
	for i := 1; ; i++ {
		record, err := csvReader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if len(record) != rowLen {
			return fmt.Errorf("error: inconsistent column length on line %d of length %d (rather than %d). ensure all csv columns same length", i, len(record), rowLen)
		}
	}
	return nil
}
