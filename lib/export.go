package lib

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/rpc"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// ExportRequests encapsulates business logic of export operation
// TODO (b5): switch to using an Instance instead of separate fields
type ExportRequests struct {
	node *p2p.QriNode
	cli  *rpc.Client
}

// CoreRequestsName implements the Requests interface
func (r ExportRequests) CoreRequestsName() string { return "export" }

// NewExportRequests creates a ExportRequests pointer from either a repo
// or an rpc.Client
func NewExportRequests(node *p2p.QriNode, cli *rpc.Client) *ExportRequests {
	if node != nil && cli != nil {
		panic(fmt.Errorf("both node and client supplied to NewExportRequests"))
	}
	return &ExportRequests{
		node: node,
		cli:  cli,
	}
}

// ExportParams defines parameters for the export method
type ExportParams struct {
	Ref       string
	TargetDir string
	Output    string
	Format    string
	Zipped    bool
}

// Export exports a dataset in the specified format
func (r *ExportRequests) Export(p *ExportParams, fileWritten *string) (err error) {
	if p.TargetDir == "" {
		p.TargetDir = "."
		if err = qfs.AbsPath(&p.TargetDir); err != nil {
			return err
		}
	}

	if r.cli != nil {
		return r.cli.Call("ExportRequests.Export", p, fileWritten)
	}

	ref := &repo.DatasetRef{}
	if p.Ref == "" {
		// Handle `qri use` to get the current default dataset.
		if err = DefaultSelectedRef(r.node.Repo, ref); err != nil {
			return err
		}
	} else {
		*ref, err = repo.ParseDatasetRef(p.Ref)
		if err != nil {
			return fmt.Errorf("'%s' is not a valid dataset reference", p.Ref)
		}
	}
	if err = repo.CanonicalizeDatasetRef(r.node.Repo, ref); err != nil {
		return err
	}

	ds, err := base.ReadDatasetPath(r.node.Repo, ref.String())
	if err != nil {
		return err
	}
	defer base.CloseDataset(ds)

	format := p.Format
	if format == "" {
		if p.Zipped {
			// Default format, if --zip flag is set, is zip
			format = "zip"
		} else {
			// Default format is json, otherwise
			format = "json"
		}
	}

	if p.Output == "" || isDirectory(p.Output) {
		// If output is blank or a directory, derive filename from repo name and commit timestamp.
		baseName, err := GenerateFilename(ds, format)
		if err != nil {
			return err
		}
		*fileWritten = path.Join(p.Output, baseName)
	} else {
		// If output filename is not blank, check that the file extension matches the format. Or
		// if format is not specified, use the file extension to derive the format.
		ext := filepath.Ext(p.Output)
		if strings.HasPrefix(ext, ".") {
			ext = ext[1:]
		}
		// If format was not supplied as a flag, and we're not outputting a zip, derive format
		// from file extension.
		if p.Format == "" && !p.Zipped {
			format = ext
		}
		// Make sure the format doesn't contradict the file extension.
		if ext != format {
			return fmt.Errorf("file extension doesn't match format %s <> %s", ext, format)
		}
		*fileWritten = p.Output
	}

	// fileWritten represents the human-readable name of where the export is written to, while
	// outputPath is an absolute path used in the implementation
	var outputPath string
	if path.IsAbs(*fileWritten) {
		outputPath = *fileWritten
	} else {
		outputPath = path.Join(p.TargetDir, *fileWritten)
	}

	// If output is a format wrapped in a zip file, fixup the output name.
	if p.Zipped && format != "zip" {
		outputPath = replaceExt(outputPath, ".zip")
		*fileWritten = replaceExt(*fileWritten, ".zip")
	}

	// Make sure output doesn't already exist.
	_, err = os.Stat(outputPath)
	if err == nil {
		return fmt.Errorf("already exists: \"%s\"", *fileWritten)
	}

	// Create output writer.
	var writer io.Writer
	writer, err = os.Create(outputPath)
	if err != nil {
		return err
	}

	// If outputting a wrapped zip file, create the zip wrapper.
	if p.Zipped && format != "zip" {
		zipWriter := zip.NewWriter(writer)

		writer, err = zipWriter.Create(fmt.Sprintf("dataset.%s", format))
		if err != nil {
			return err
		}

		defer func() {
			zipWriter.Close()
		}()
	}

	// Create entry reader.
	reader, err := dsio.NewEntryReader(ds.Structure, ds.BodyFile())
	if err != nil {
		return err
	}

	switch format {
	case "json":

		// TODO (dlong): Look into combining this functionality (reading body, changing structure),
		// and moving it into a method of `actions`.
		bodyEntries, err := base.ReadEntries(reader)
		if err != nil {
			return err
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
			return err
		}
		return nil

	case "yaml":

		bodyEntries, err := base.ReadEntries(reader)
		if err != nil {
			return err
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
			return err
		}

		_, err = writer.Write(dsBytes)
		if err != nil {
			return err
		}
		return nil

	case "xlsx":
		st := &dataset.Structure{
			Format: "xlsx",
			// FormatConfig: map[string]interface{}{
			// 	"sheetName": "body",
			// },
		}
		w, err := dsio.NewEntryWriter(st, writer)
		if err != nil {
			return err
		}

		if err := dsio.Copy(reader, w); err != nil {
			return err
		}
		return w.Close()

	case "zip":

		store := r.node.Repo.Store()
		if err = dsutil.WriteZipArchive(store, ds, "json", ref.String(), writer); err != nil {
			return err
		}

		return nil

	default:
		return fmt.Errorf("unknown file format \"%s\"", format)
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
