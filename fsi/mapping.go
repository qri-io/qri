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
func ReadDir(dir string) (ds *dataset.Dataset, mapping map[string]string, err error) {
	mapping = map[string]string{}
	ds = &dataset.Dataset{}
	schema := map[string]interface{}{}

	components := map[string]interface{}{
		componentNameDataset: ds,

		componentNameCommit:    &dataset.Commit{},
		componentNameMeta:      &dataset.Meta{},
		componentNameStructure: &dataset.Structure{},
		componentNameSchema:    &schema,
		componentNameTransform: &dataset.Transform{},
		componentNameViz:       &dataset.Viz{},

		// TODO (b5) - deal with dataset bodies
		componentNameBody: nil,
	}

	extensions := map[string]decoderFactory{
		".json": newJSONDecoder,
		".yaml": newYAMLDecoder,
		".yml":  newYAMLDecoder,
	}

	addMapping := func(cmpName, path string) error {
		if cmpPath, exists := mapping[cmpName]; exists {
			cmpPath = filepath.Base(cmpPath)
			path = filepath.Base(path)
			return fmt.Errorf(`%s is defined in two places: %s and %s. please remove one`, cmpName, cmpPath, path)
		}

		mapping[cmpName] = path
		return nil
	}

	for cmpName, cmp := range components {
		for ext, mkDec := range extensions {
			filename := fmt.Sprintf("%s%s", cmpName, ext)
			path := filepath.Join(dir, filename)
			if f, e := os.Open(path); e == nil {
				if cmpName != componentNameBody {
					if err = mkDec(f).Decode(cmp); err != nil {
						err = fmt.Errorf("reading %s: %s", filename, err)
						return ds, mapping, err
					}
				}

				if err = addMapping(cmpName, path); err != nil {
					return ds, mapping, err
				}

				switch cmpName {
				case componentNameDataset:
					if ds.Commit != nil {
						if err = addMapping(componentNameCommit, path); err != nil {
							return
						}
					}
					if ds.Meta != nil {
						if err = addMapping(componentNameMeta, path); err != nil {
							return
						}
					}
					if ds.Structure != nil {
						if err = addMapping(componentNameStructure, path); err != nil {
							return
						}
						if ds.Structure.Schema != nil {
							if err = addMapping(componentNameSchema, path); err != nil {
								return
							}
						}
					}
					if ds.Viz != nil {
						if err = addMapping(componentNameViz, path); err != nil {
							return
						}
					}
					if ds.Transform != nil {
						if err = addMapping(componentNameTransform, path); err != nil {
							return
						}
					}
					if ds.Body != nil {
						if err = addMapping(componentNameBody, path); err != nil {
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
				case componentNameViz:
					ds.Viz = cmp.(*dataset.Viz)
				case componentNameTransform:
					ds.Transform = cmp.(*dataset.Transform)
				case componentNameBody:
						ds.BodyPath = path
					// 	ds.Body = cmp.(*dataset.Body)
					// 	// TODO (b5) -
				}
			}
		}
	}

	if len(mapping) == 0 {
		err = ErrNoDatasetFiles
	}

	return ds, mapping, err
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
