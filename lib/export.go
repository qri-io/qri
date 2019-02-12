package lib

import (
	"encoding/json"
	"fmt"
	"net/rpc"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/dataset/dsutil"
	"github.com/qri-io/qri/base"
	"github.com/qri-io/qri/p2p"
	"github.com/qri-io/qri/repo"
)

// ExportRequests encapsulates business logic of export operation
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
}

// TODO (dlong): Tests!

// Export exports a dataset in the specified format
func (r *ExportRequests) Export(p *ExportParams, fileWritten *string) (err error) {
	if p.TargetDir == "" {
		p.TargetDir = "."
		if err = AbsPath(&p.TargetDir); err != nil {
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
		// Default format is json
		format = "json"
	}

	if p.Output == "" || isDirectory(p.Output) {
		// If output is blank or a directory, derive filename from repo name and commit timestamp.
		ts := ds.Commit.Timestamp
		timeText := fmt.Sprintf("%04d-%02d-%02d-%02d-%02d-%02d", ts.Year(), ts.Month(), ts.Day(),
			ts.Hour(), ts.Minute(), ts.Second())
		baseName := fmt.Sprintf("%s-%s_-_%s.%s", ds.Peername, ds.Name, timeText, format)
		*fileWritten = path.Join(p.Output, baseName)
	} else {
		// If output filename is not blank, check that the file extension matches the format. Or
		// if format is not specified, use the file extension to derive the format.
		ext := filepath.Ext(p.Output)
		if strings.HasPrefix(ext, ".") {
			ext = ext[1:]
		}

		if p.Format == "" {
			format = ext
		}

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

	_, err = os.Stat(outputPath)
	if err == nil {
		return fmt.Errorf("already exists: \"%s\"", *fileWritten)
	}

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

		dst, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		if err := json.NewEncoder(dst).Encode(ds); err != nil {
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

		dst, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		_, err = dst.Write(dsBytes)
		if err != nil {
			return err
		}
		return nil

	case "xlsx":
		f, err := os.Create(outputPath)
		if err != nil {
			return err
		}

		st := &dataset.Structure{
			Format: "xlsx",
			// FormatConfig: map[string]interface{}{
			// 	"sheetName": "body",
			// },
		}
		w, err := dsio.NewEntryWriter(st, f)
		if err != nil {
			return err
		}

		if err := dsio.Copy(reader, w); err != nil {
			return err
		}
		return w.Close()

	case "zip":
		// default to a zip archive
		w, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		store := r.node.Repo.Store()
		if err = dsutil.WriteZipArchive(store, ds, "json", ref.String(), w); err != nil {
			return err
		}

		return w.Close()

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
