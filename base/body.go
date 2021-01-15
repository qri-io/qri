package base

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qfs"
)

// ErrNoBodyToInline is an error returned when a dataset has no body for inlining
var ErrNoBodyToInline = errors.New("no body to inline")

// ReadBody grabs some or all of a dataset's body, writing an output in the desired format
func ReadBody(ds *dataset.Dataset, format dataset.DataFormat, fcfg dataset.FormatConfig, limit, offset int, all bool) (data []byte, err error) {
	if ds == nil {
		return nil, fmt.Errorf("can't load body from a nil dataset")
	}

	file := ds.BodyFile()
	if file == nil {
		err = fmt.Errorf("no body file to read")
		return
	}

	st := &dataset.Structure{}
	assign := &dataset.Structure{
		Format: format.String(),
		Schema: ds.Structure.Schema,
	}
	if fcfg != nil {
		assign.FormatConfig = fcfg.Map()
	}
	st.Assign(ds.Structure, assign)

	data, err = ConvertBodyFile(file, ds.Structure, st, limit, offset, all)
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	return data, nil
}

// ReadEntries reads entries and returns them as a native go array or map
func ReadEntries(reader dsio.EntryReader) (interface{}, error) {
	obj := make(map[string]interface{})
	array := make([]interface{}, 0)

	tlt, err := dsio.GetTopLevelType(reader.Structure())
	if err != nil {
		return nil, err
	}

	for i := 0; ; i++ {
		val, err := reader.ReadEntry()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		if tlt == "object" {
			obj[val.Key] = val.Value
		} else {
			array = append(array, val.Value)
		}
	}

	if tlt == "object" {
		return obj, nil
	}
	return array, nil
}

// InlineJSONBody reads the contents dataset.BodyFile() into a json.RawMessage,
// assigning the result to dataset.Body
func InlineJSONBody(ds *dataset.Dataset) error {
	file := ds.BodyFile()
	if file == nil {
		return ErrNoBodyToInline
	}

	if ds.Structure.Format == dataset.JSONDataFormat.String() {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		ds.Body = json.RawMessage(data)
		return nil
	}

	in := ds.Structure
	st := &dataset.Structure{}
	st.Assign(in, &dataset.Structure{
		Format: "json",
		Schema: in.Schema,
	})

	data, err := ConvertBodyFile(file, in, st, 0, 0, true)
	if err != nil {
		log.Errorf("converting body file to JSON: %s", err)
		return fmt.Errorf("converting body file to JSON: %s", err)
	}

	ds.Body = json.RawMessage(data)
	return nil
}

// ConvertBodyFile takes an input file & structure, and converts a specified selection
// to the structure specified by out
func ConvertBodyFile(file qfs.File, in, out *dataset.Structure, limit, offset int, all bool) (data []byte, err error) {
	buf := &bytes.Buffer{}

	w, err := dsio.NewEntryWriter(out, buf)
	if err != nil {
		return
	}

	// TODO(dlong): Kind of a hacky one-off. Generalize this for other format options.
	if out.DataFormat() == dataset.JSONDataFormat {
		ok, pretty := out.FormatConfig["pretty"].(bool)
		if ok && pretty {
			w, err = dsio.NewJSONPrettyWriter(out, buf, " ")
		}
	}
	if err != nil {
		return
	}

	rr, err := dsio.NewEntryReader(in, file)
	if err != nil {
		err = fmt.Errorf("error allocating data reader: %s", err)
		return
	}

	if !all {
		rr = &dsio.PagedReader{
			Reader: rr,
			Limit:  limit,
			Offset: offset,
		}
	}
	err = dsio.Copy(rr, w)

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("error closing row buffer: %s", err.Error())
	}

	return buf.Bytes(), nil
}

// ConvertBodyFormat rewrites a body from a source format to a destination format.
// TODO (b5): Combine this with ConvertBodyFile, update callers.
func ConvertBodyFormat(bodyFile qfs.File, fromSt, toSt *dataset.Structure) (qfs.File, error) {
	// Reader for entries of the source body.
	r, err := dsio.NewEntryReader(fromSt, bodyFile)
	if err != nil {
		return nil, err
	}

	// Writes entries to a new body.
	buffer := &bytes.Buffer{}
	w, err := dsio.NewEntryWriter(toSt, buffer)
	if err != nil {
		return nil, err
	}

	err = dsio.Copy(r, w)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}

	return qfs.NewMemfileReader(fmt.Sprintf("body.%s", toSt.Format), buffer), nil
}
