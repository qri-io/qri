package base

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/qri-io/qfs"

	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsviz"
	"github.com/qri-io/qri/repo"
)

// Render executes a template for a dataset, returning a slice of HTML
func Render(r repo.Repo, ref repo.DatasetRef, tmplData []byte, limit, offset int, all bool) ([]byte, error) {
	const tmplName = "template"
	var rdr io.Reader

	err := repo.CanonicalizeDatasetRef(r, &ref)
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}

	store := r.Store()

	ds, err := dsfs.LoadDataset(store, ref.Path)
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}
	if err := OpenDataset(r.Filesystem(), ds); err != nil {
		return nil, err
	}

	if tmplData != nil {
		rdr = bytes.NewBuffer(tmplData)
	}

	// TODO - hack for now. a subpackage of dataset should handle all of the below,
	// and use a method to set the default template if one can be loaded from the web
	if rdr == nil && ds.Viz != nil && ds.Viz.ScriptPath != "" {
		f, err := store.Get(ds.Viz.ScriptPath)
		if err != nil {
			return nil, fmt.Errorf("loading template from store: %s", err.Error())
		}
		rdr = f
	}

	if rdr == nil {
		rdr = strings.NewReader(DefaultTemplate)
	}

	ds.Viz.Format = "html"
	ds.Viz.SetScriptFile(qfs.NewMemfileReader("viz.html", rdr))

	data, err := dsviz.Render(ds)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(data)

	// tmplBytes, err := ioutil.ReadAll(rdr)
	// if err != nil {
	// 	return nil, fmt.Errorf("reading template data: %s", err.Error())
	// }

	// tmpl, err := template.New(tmplName).Parse(string(tmplBytes))
	// if err != nil {
	// 	return nil, fmt.Errorf("parsing template: %s", err.Error())
	// }

	// file, err := dsfs.LoadBody(store, ds)
	// if err != nil {
	// 	log.Debug(err.Error())
	// 	return nil, err
	// }

	// rr, err := dsio.NewEntryReader(ds.Structure, file)
	// if err != nil {
	// 	return nil, fmt.Errorf("error allocating data reader: %s", err)
	// }

	// if !all {
	// 	rr = &dsio.PagedReader{Reader: rr, Limit: limit, Offset: offset}
	// }
	// bodyEntries, err := ReadEntries(rr)
	// if err != nil {
	// 	return nil, err
	// }

	// // TODO (b5): repo.DatasetRef should be refactored into this newly expanded DatasetPod,
	// // once that's done these values should be populated by ds.Encode(), removing the need
	// // for these assignments
	// ds.Peername = ref.Peername
	// ds.ProfileID = ref.ProfileID.String()
	// ds.Name = ref.Name
	// if ds.Meta == nil {
	// 	ds.Meta = &dataset.Meta{}
	// }

	// ds.Body = bodyEntries

	// tmplBuf := &bytes.Buffer{}
	// if err := tmpl.Execute(tmplBuf, ds); err != nil {
	// 	return nil, err
	// }
	// return tmplBuf.Bytes(), nil
}

// DefaultTemplate is the template that render will fall back to should no
// template be available
var DefaultTemplate = `<!DOCTYPE html>
<html>
<head>
  <title>{{ .Peername }}/{{ .Name }}</title>
  <style type="text/css">
    html, body, .viewport { height: 100%; width: 100%; margin: 0; font-family: "avenir next", "avenir", sans-serif; font-size: 16px; }
    body { display: flex; flex-direction: column; }
    header { width: 100%; background: #0061A6; color: white; padding: 80px 0 40px 0; }
    section { min-height: 450px; }
    footer { width: 100%; background: #EBEBEB; padding: 40px 0 20px 0; margin-top: 40px; }
    header, section, footer { width: 100%; flex: 1; }
    label { display: block; font-weight: normal; color: #999; text-transform: uppercase; font-size: 14px; }
    .content { margin: 0 auto; max-width: 600px; }
    .stat { font-weight: bold; }
    .ref { margin-top: 5px; }}
    .path { color: #bebebe; }
  </style>
</head>
<body class="viewport">
  <header>
    <div class="content">
      <label style="color: white">Dataset</label>
      <h4 class="ref">{{.Peername}}/{{ .Name }}</h4>
      <h1>{{ .Meta.Title }}</h1>
      <small class="path">{{ .Path }}</small>
      <p>{{ .Meta.Description }}</p>
    </div>
  </header>
  <section>
    <div class="content">
      <p class="stat"><label>updated:</label>{{ .Commit.Timestamp.Format "Mon, 02 Jan 2006" }}</p>
      <p class="stat"><label>data format:</label>{{ .Structure.Format }}</p>
      <p class="stat"><label>entry count:</label>{{ .Structure.Entries }}</p>
      <p class="stat"><label>errors:</label>{{ .Structure.ErrCount }}</p>
      <p class="stat"><label>commit title:</label>{{ .Commit.Title }}</p>
    </div>
  </section>
  <footer>
    <div class="content">
      {{ if .Meta.License }}
        <p>License: <a href="{{ .Meta.License.URL }}">{{ .Meta.License.Type }}</a></p>
      {{ end }}
      <p>Created with <a href="https://qri.io">qri</a></p>
    </div>
  </footer>
</body>
</html>`
