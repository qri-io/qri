package validate

import (
	"fmt"
	"github.com/datatogether/cdxj"
	"io"
	"regexp"
	"strconv"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/datatypes"
	"github.com/qri-io/dataset/dsio"
)

var alphaNumericRegex = regexp.MustCompile(`^[a-zA-Z]\w{0,143}$`)

// DataFormat ensures that for each accepted dataset.DataFormat,
// we havea well-formed dataset (eg. for csv, we need rows to all
// be of same length)
func DataFormat(df dataset.DataFormat, r io.Reader) error {
	switch df {
	// explicitly supported at present
	case dataset.CsvDataFormat:
		return CheckCsvRowLengths(r)
	case dataset.CdxjDataFormat:
		return cdxj.Validate(r)
	// explicitly unsupported at present
	case dataset.JsonDataFormat:
		return fmt.Errorf("error: data format 'JsonData' not currently supported")
	case dataset.XlsDataFormat:
		return fmt.Errorf("error: data format 'XlsData' not currently supported")
	case dataset.XmlDataFormat:
		return fmt.Errorf("error: data format 'XmlData' not currently supported")
	// *implicitly unsupported
	case dataset.UnknownDataFormat:
		return fmt.Errorf("error: unknown data format not currently supported")
	default:
		return fmt.Errorf("error: data format not currently supported")
	}
}

const (
	ErrFmtOneHotMatrix = "oneHotMatrix"
	ErrFmtString       = "string"
)

type DataErrorsCfg struct {
	ErrorFormat string
	// DataFormat  DataFormat
}

func DefaultDataErrorsCfg() *DataErrorsCfg {
	return &DataErrorsCfg{
		ErrorFormat: ErrFmtString,
	}
}

// DataErrors generates a new dataset that represents data errors with the passed in dataset reader
func DataErrors(r dsio.RowReader, options ...func(*DataErrorsCfg)) (errors dsio.RowReader, count int, err error) {
	cfg := DefaultDataErrorsCfg()
	for _, opt := range options {
		opt(cfg)
	}

	vst := &dataset.Structure{
		Format: dataset.CsvDataFormat,
		Schema: &dataset.Schema{
			Fields: []*dataset.Field{
				{Name: "row_index", Type: datatypes.Integer},
			},
		},
	}
	for _, f := range r.Structure().Schema.Fields {
		vst.Schema.Fields = append(vst.Schema.Fields, &dataset.Field{Name: f.Name + "_error", Type: datatypes.String})
	}

	buf := dsio.NewBuffer(vst)

	err = dsio.EachRow(r, func(num int, row [][]byte, err error) error {
		if err != nil {
			return err
		}

		errData, errNum, err := validateRow(r.Structure().Schema.Fields, num, row)
		if err != nil {
			return err
		}

		count += errNum
		if errNum != 0 {
			buf.WriteRow(errData)
		}

		return nil
	})
	if err != nil {
		return
	}

	if err = buf.Close(); err != nil {
		err = fmt.Errorf("error closing valdation buffer: %s", err.Error())
		return
	}

	errors = buf
	return
}

func validateRow(fields []*dataset.Field, num int, row [][]byte) ([][]byte, int, error) {
	count := 0
	errors := make([][]byte, len(fields)+1)
	errors[0] = []byte(strconv.FormatInt(int64(num), 10))
	if len(row) != len(fields) {
		return errors, count, fmt.Errorf("column mismatch. expected: %d, got: %d", len(fields), len(row))
	}

	for i, f := range fields {
		_, e := f.Type.Parse(row[i])
		if e != nil {
			count++
			errors[i+1] = []byte(e.Error())
		} else {
			errors[i+1] = []byte("")
		}
	}

	return errors, count, nil
}

// func (ds *Resource) ValidateDeadLinks(store fs.Store) (validation *Resource, data []byte, count int, err error) {
// 	proj := map[int]int{}
// 	validation = &Resource{
// 		Address: NewAddress(ds.Address.String(), "errors"),
// 		Format:  CsvDataFormat,
// 	}

// 	for i, f := range ds.Fields {
// 		if f.Type == datatype.Url {
// 			proj[i] = len(validation.Fields)
// 			validation.Fields = append(validation.Fields, f)
// 			validation.Fields = append(validation.Fields, &Field{Name: f.Name + "_dead", Type: datatype.Integer})
// 		}
// 	}

// 	dsData, e := ds.FetchBytes(store)
// 	if e != nil {
// 		err = e
// 		return
// 	}
// 	ds.Data = dsData

// 	buf := &bytes.Buffer{}
// 	cw := csv.NewWriter(buf)

// 	err = ds.EachRow(func(num int, row [][]byte, err error) error {
// 		if err != nil {
// 			return err
// 		}

// 		result := make([][]byte, len(validation.Fields))
// 		for l, r := range proj {
// 			result[r] = row[l]
// 			if err := checkUrl(string(result[r])); err != nil {
// 				count++
// 				result[r+1] = []byte("1")
// 			} else {
// 				result[r+1] = []byte("0")
// 			}
// 		}

// 		csvRow := make([]string, len(result))
// 		for i, d := range result {
// 			csvRow[i] = string(d)
// 		}
// 		if err := cw.Write(csvRow); err != nil {
// 			fmt.Sprintln(err)
// 		}

// 		return nil
// 	})

// 	cw.Flush()
// 	data = buf.Bytes()
// 	return
// }

// func checkUrl(rawurl string) error {
// 	u, err := url.Parse(rawurl)
// 	if err != nil {
// 		fmt.Println(err)
// 		return err
// 	}
// 	if u.Scheme == "" {
// 		u.Scheme = "http"
// 	}
// 	res, err := http.Get(u.String())
// 	if err != nil {
// 		return err
// 	}
// 	res.Body.Close()
// 	fmt.Println(u.String(), res.StatusCode)
// 	if res.StatusCode > 399 {
// 		return fmt.Errorf("non-200 status code")
// 	}
// 	return nil
// }
