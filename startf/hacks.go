package startf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qfs"
)

// InlineJSONBody reads the contents dataset.BodyFile() into a json.RawMessage,
// assigning the result to dataset.Body
// TODO (b5) - this is copied from base to avoid an import cycle. we should get
// base to not rely on the startf package so we can import this from base
func InlineJSONBody(ds *dataset.Dataset) error {
	file := ds.BodyFile()
	if file == nil {
		log.Error("no body file")
		return fmt.Errorf("no response body file")
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
// TODO (b5) - this is copied from base to avoid an import cycle. we should get
// base to not rely on the startf package so we can import this from base
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
