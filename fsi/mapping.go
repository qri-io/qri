package fsi

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dataset"
)

const (
	componentNameCommit    = "commit"
	componentNameDataset   = "dataset"
	componentNameMeta      = "meta"
	componentNameSchema    = "schema"
	componentNameBody      = "body"
	componentNameStructure = "structure"
	componentNameTransform = "transform"
	componentNameViz       = "viz"
)

var (
	// ErrNoDatasetFiles indicates no data
	ErrNoDatasetFiles = fmt.Errorf("no dataset files provided")
)

// ReadDir parses a directory into a dataset, returning both the dataset and
// a map of component names to the files they came from. Files can be specified
// in either JSON or YAML format. It is an error to specify any component more
// than once
func ReadDir(dir string) (ds *dataset.Dataset, fileMap map[string]string, problems map[string]string, err error) {
	fileMap = map[string]string{}
	ds = &dataset.Dataset{}
	schema := map[string]interface{}{}
	problems = nil

	components := map[string]interface{}{
		componentNameDataset: ds,

		componentNameCommit:    &dataset.Commit{},
		componentNameMeta:      &dataset.Meta{},
		componentNameStructure: &dataset.Structure{},
		componentNameSchema:    &schema,
		componentNameTransform: &dataset.Transform{},
		componentNameViz:       &dataset.Viz{},

		componentNameBody: nil,
	}

	extensions := map[string]decoderFactory{
		".json": newJSONDecoder,
		".yaml": newYAMLDecoder,
		".yml":  newYAMLDecoder,
	}

	addFile := func(cmpName, path string) error {
		if cmpPath, exists := fileMap[cmpName]; exists {
			cmpPath = filepath.Base(cmpPath)
			path = filepath.Base(path)
			return fmt.Errorf(`%s is defined in two places: %s and %s. please remove one`, cmpName, cmpPath, path)
		}

		fileMap[cmpName] = path
		return nil
	}

	// HACK: Detect body format.
	bodyFormat := ""
	if _, err = os.Stat(filepath.Join(dir, "body.csv")); !os.IsNotExist(err) {
		bodyFormat = "csv"
	}
	if _, err = os.Stat(filepath.Join(dir, "body.json")); !os.IsNotExist(err) {
		if bodyFormat == "csv" {
			return ds, fileMap, problems, fmt.Errorf("body.csv and body.json both exist")
		}
		bodyFormat = "json"
	}

	bodyFilename := ""
	if bodyFormat != "" {
		bodyFilename = fmt.Sprintf("body.%s", bodyFormat)
		if err = addFile(componentNameBody, bodyFilename); err != nil {
			return ds, fileMap, problems, err
		}
		if ds.BodyPath == "" {
			ds.BodyPath = filepath.Join(dir, bodyFilename)
		}
	}

	// Iterate components in a deterministic order, from highest priority to lowest.
	for i := 0; i < len(componentListOrder); i++ {
		cmpName := componentListOrder[i]
		cmp := components[cmpName]
		for ext, mkDec := range extensions {
			filename := fmt.Sprintf("%s%s", cmpName, ext)
			path := filepath.Join(dir, filename)
			if f, e := os.Open(path); e == nil {
				if cmpName != componentNameBody {
					if err = mkDec(f).Decode(cmp); err != nil {
						if problems == nil {
							problems = make(map[string]string)
						}
						problems[cmpName] = filename
						continue
					}
					if err = addFile(cmpName, path); err != nil {
						return ds, fileMap, problems, err
					}
				}

				switch cmpName {
				case componentNameDataset:
					if ds.Commit != nil {
						if err = addFile(componentNameCommit, path); err != nil {
							return
						}
					}
					if ds.Meta != nil {
						if err = addFile(componentNameMeta, path); err != nil {
							return
						}
					}
					if ds.Structure != nil {
						if err = addFile(componentNameStructure, path); err != nil {
							return
						}
						if ds.Structure.Schema != nil {
							if err = addFile(componentNameSchema, path); err != nil {
								return
							}
						}
					}
					if ds.Viz != nil {
						if err = addFile(componentNameViz, path); err != nil {
							return
						}
					}
					if ds.Transform != nil {
						if err = addFile(componentNameTransform, path); err != nil {
							return
						}
					}
					if ds.Body != nil {
						if err = addFile(componentNameBody, path); err != nil {
							return
						}
					}

				case componentNameCommit:
					ds.Commit = cmp.(*dataset.Commit)
				case componentNameMeta:
					ds.Meta = cmp.(*dataset.Meta)
				case componentNameStructure:
					ds.Structure = cmp.(*dataset.Structure)
				case componentNameSchema:
					if ds.Structure == nil {
						ds.Structure = &dataset.Structure{}
					}
					ds.Structure.Schema = *cmp.(*map[string]interface{})
					if ds.Structure.Format == "" {
						ds.Structure.Format = bodyFormat
					}
				case componentNameViz:
					ds.Viz = cmp.(*dataset.Viz)
				case componentNameTransform:
					ds.Transform = cmp.(*dataset.Transform)
				case componentNameBody:
					if ds.BodyPath == "" {
						ds.BodyPath = path
					}
				}
			}
		}
	}

	if len(fileMap) == 0 {
		err = ErrNoDatasetFiles
	}

	return ds, fileMap, problems, err
}

type decoderFactory func(io.Reader) decoder

type decoder interface {
	Decode(m interface{}) error
}

type jsonDecoder struct {
	dec *json.Decoder
}

func newJSONDecoder(r io.Reader) decoder {
	return jsonDecoder{
		dec: json.NewDecoder(r),
	}
}

func (jd jsonDecoder) Decode(v interface{}) error {
	return jd.dec.Decode(v)
}

type yamlDecoder struct {
	rdr io.Reader
}

func newYAMLDecoder(r io.Reader) decoder {
	return yamlDecoder{
		rdr: r,
	}
}

func (yd yamlDecoder) Decode(v interface{}) error {
	// convert yaml input to json as a hack to support yaml input for now
	yamlData, err := ioutil.ReadAll(yd.rdr)
	if err != nil {
		return fmt.Errorf("invalid file: %s", err.Error())
	}

	jsonData, err := yaml.YAMLToJSON(yamlData)
	if err != nil {
		return fmt.Errorf("converting yaml body to json: %s", err.Error())
	}

	return json.Unmarshal(jsonData, v)
}
