package detect

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/qri-io/dataset"
)

var (
	spaces         = regexp.MustCompile(`[\s-]+`)
	nonAlpha       = regexp.MustCompile(`[^a-zA-z0-9_]`)
	carriageReturn = regexp.MustCompile(`(?m)\r[^\n]`)
)

// FromFile takes a filepath & tries to work out the corresponding dataset
// for the sake of speed, it only works with files that have a recognized extension
func FromFile(path string) (ds *dataset.Structure, err error) {
	// if filepath.Base(path) == dataset.Filename {
	// 	return nil, fmt.Errorf("cannot determine schema of a %s file", dataset.Filename)
	// }

	format, err := ExtensionDataFormat(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return DetectStructure(format, f)
}

// FromReader is a shorthand for a path/filename and reader
func FromReader(path string, data io.Reader) (ds *dataset.Structure, err error) {
	format, err := ExtensionDataFormat(path)
	if err != nil {
		return nil, err
	}
	return DetectStructure(format, data)
}

// Structure attemptes to extract a reader based on a given format and data reader
func DetectStructure(format dataset.DataFormat, data io.Reader) (r *dataset.Structure, err error) {

	r = &dataset.Structure{
		Format: format,
	}

	// ds.Data = ReplaceSoloCarriageReturns(ds.Data)
	r.Schema = &dataset.Schema{}

	r.Schema.Fields, err = Fields(r, data)
	if err != nil {
		return
	}

	return
}

// Camelize creates a valid address name (using underscores in place of any spaces)
func Camelize(path string) (name string) {
	name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	name = spaces.ReplaceAllString(name, "_")
	name = nonAlpha.ReplaceAllString(name, "")
	name = strings.Trim(name, "_")
	name = strings.ToLower(name)
	if startsWithNumberRegex.MatchString(name) {
		name = "_" + name
	}
	return
}

// ReplaceSoloCarriageReturns looks for instances of lonely \r replacing them with \r\n
// lots of files in the wild will come without "proper" line breaks, which irritates go's
// native csv package.
// TODO - make software robust to the problem, instead presenting a warning to the user
// also, we should write all output files with unified line breaks.
func ReplaceSoloCarriageReturns(data []byte) []byte {
	if carriageReturn.Match(data) {
		return carriageReturn.ReplaceAllFunc(data, func(in []byte) []byte {
			return []byte{'\r', '\n', in[1]}
		})
	}

	return data
}

// DataFormat does it's best to determine the format of a specified dataset
// func DataFormat(path string) (format dataset.DataFormat, err error) {
// 	return ExtensionDataFormat(path)
// }

// ExtensionDataFormat returns the corresponding DataFormat for a given file extension if one exists
// TODO - this should probably come from the dataset package
func ExtensionDataFormat(path string) (format dataset.DataFormat, err error) {
	ext := filepath.Ext(path)
	switch ext {
	case ".csv":
		return dataset.CsvDataFormat, nil
	case ".json":
		return dataset.JsonDataFormat, nil
	case ".xml":
		return dataset.XmlDataFormat, nil
	case ".xls":
		return dataset.XlsDataFormat, nil
	case "":
		return dataset.UnknownDataFormat, errors.New("no file extension provided")
	default:
		return dataset.UnknownDataFormat, fmt.Errorf("unrecognized file extension: '%s' ", ext)
	}

	// can't happen
	return
}
