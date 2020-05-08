// Package archive creates and consumes high-fidelity conversions of dataset
// documents for export & import
package archive

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	logger "github.com/ipfs/go-log"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qfs/cafs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/dsref"
)

var log = logger.Logger("archive")

// Export generates a high-fidelity copy of a dataset that doesn't require qri
// software to read.
// TODO (b5) - this currently has a lot of overlap with "get" and "checkout"
// commands, we should emphasize those (more common) tools instead. See
// https://github.com/qri-io/qri/issues/1176 for discussion
func Export(ctx context.Context, store cafs.Filestore, ds *dataset.Dataset, refStr, targetDir, output, outputFormat string, zipped bool) (string, error) {
	var err error
	defer base.CloseDataset(ds)

	format := outputFormat
	if format == "" {
		if zipped {
			// Default format, if --zip flag is set, is zip
			format = "zip"
		} else {
			// Default format is json, otherwise
			format = "json"
		}
	}

	var fileWritten string
	if output == "" || isDirectory(output) {
		// If output is blank or a directory, derive filename from repo name and commit timestamp.
		baseName, err := GenerateFilename(ds, format)
		if err != nil {
			return "", err
		}
		fileWritten = path.Join(output, baseName)
	} else {
		// If output filename is not blank, check that the file extension matches the format. Or
		// if format is not specified, use the file extension to derive the format.
		ext := filepath.Ext(output)
		if strings.HasPrefix(ext, ".") {
			ext = ext[1:]
		}
		// If format was not supplied as a flag, and we're not outputting a zip, derive format
		// from file extension.
		if outputFormat == "" && !zipped {
			format = ext
		}
		// Make sure the format doesn't contradict the file extension.
		if ext != format {
			return "", fmt.Errorf("file extension doesn't match format %s <> %s", ext, format)
		}
		fileWritten = output
	}

	// fileWritten represents the human-readable name of where the export is written to, while
	// outputPath is an absolute path used in the implementation
	var outputPath string
	if path.IsAbs(fileWritten) {
		outputPath = fileWritten
	} else {
		outputPath = path.Join(targetDir, fileWritten)
	}

	// If output is a format wrapped in a zip file, fixup the output name.
	if zipped && format != "zip" {
		outputPath = replaceExt(outputPath, ".zip")
		fileWritten = replaceExt(fileWritten, ".zip")
	}

	// Make sure output doesn't already exist.
	if _, err = os.Stat(outputPath); err == nil {
		return "", fmt.Errorf(`already exists: "%s"`, fileWritten)
	}

	// Create output writer.
	var writer io.Writer
	writer, err = os.Create(outputPath)
	if err != nil {
		return "", err
	}

	// If outputting a wrapped zip file, create the zip wrapper.
	if zipped && format != "zip" {
		zipWriter := zip.NewWriter(writer)

		writer, err = zipWriter.Create(fmt.Sprintf("dataset.%s", format))
		if err != nil {
			return "", err
		}

		defer func() {
			zipWriter.Close()
		}()
	}

	// Create entry reader.
	reader, err := dsio.NewEntryReader(ds.Structure, ds.BodyFile())
	if err != nil {
		return "", err
	}

	switch format {
	case "json":

		// TODO (dlong): Look into combining this functionality (reading body, changing structure),
		// with some of the functions in `base`.
		bodyEntries, err := base.ReadEntries(reader)
		if err != nil {
			return "", err
		}
		ds.Body = bodyEntries

		ds.Structure = &dataset.Structure{
			Format:   "json",
			Schema:   ds.Structure.Schema,
			Depth:    ds.Structure.Depth,
			ErrCount: ds.Structure.ErrCount,
		}
		// drop any transform stuff
		ds.Transform = nil

		if err := json.NewEncoder(writer).Encode(ds); err != nil {
			return "", err
		}
		return fileWritten, nil

	case "yaml":

		bodyEntries, err := base.ReadEntries(reader)
		if err != nil {
			return "", err
		}
		ds.Body = bodyEntries

		ds.Structure = &dataset.Structure{
			Format:   "yaml",
			Schema:   ds.Structure.Schema,
			Depth:    ds.Structure.Depth,
			ErrCount: ds.Structure.ErrCount,
		}
		// drop any transform stuff
		ds.Transform = nil
		dsBytes, err := yaml.Marshal(ds)
		if err != nil {
			return "", err
		}

		_, err = writer.Write(dsBytes)
		if err != nil {
			return "", err
		}
		return fileWritten, nil

	case "xlsx":
		st := &dataset.Structure{
			Format: "xlsx",
			Schema: ds.Structure.Schema,
		}
		w, err := dsio.NewEntryWriter(st, writer)
		if err != nil {
			return "", err
		}

		if err := dsio.Copy(reader, w); err != nil {
			return "", err
		}
		return fileWritten, w.Close()

	case "zip":
		ref, err := dsref.Parse(refStr)
		if err != nil {
			return "", err
		}
		blankInitID := ""
		if err = WriteZip(ctx, store, ds, "json", blankInitID, ref, writer); err != nil {
			return "", err
		}

		return fileWritten, nil

	default:
		return "", fmt.Errorf("unknown file format \"%s\"", format)
	}
}

func isDirectory(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return st.Mode().IsDir()
}

func replaceExt(filename, newExt string) string {
	ext := path.Ext(filename)
	return filename[:len(filename)-len(ext)] + newExt
}

// GenerateFilename takes a dataset and generates a filename
// if no timestamp exists, it will default to the empty time.Time
// in the form [peername]-[datasetName]_-_[timestamp].[format]
func GenerateFilename(ds *dataset.Dataset, format string) (string, error) {
	ts := time.Time{}
	if ds.Commit != nil {
		ts = ds.Commit.Timestamp
	}
	if format == "" {
		if ds.Structure == nil || ds.Structure.Format == "" {
			return "", fmt.Errorf("no format specified and no format present in the dataset Structure")
		}
		format = ds.Structure.Format
	}
	timeText := fmt.Sprintf("%04d-%02d-%02d-%02d-%02d-%02d", ts.Year(), ts.Month(), ts.Day(),
		ts.Hour(), ts.Minute(), ts.Second())
	return fmt.Sprintf("%s-%s_-_%s.%s", ds.Peername, ds.Name, timeText, format), nil
}
