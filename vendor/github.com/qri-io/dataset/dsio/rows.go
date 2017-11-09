package dsio

import (
	"fmt"
	"io"
)

// DataIteratorFunc is a function for each "row" of a resource's raw data
type DataIteratorFunc func(int, [][]byte, error) error

// EachRow calls fn on each row of a given RowReader
func EachRow(rr RowReader, fn DataIteratorFunc) error {
	num := 0
	for {
		row, err := rr.ReadRow()
		if err != nil {
			if err.Error() == io.EOF.Error() {
				return nil
			}
			return fmt.Errorf("error reading row: %s", err.Error())
		}

		if err := fn(num, row, err); err != nil {
			if err.Error() == io.EOF.Error() {
				return nil
			}
			return err
		}
		num++
	}

	return fmt.Errorf("cannot parse data format '%s'", rr.Structure().Format.String())
}
