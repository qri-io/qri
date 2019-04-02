package base

import (
	"io/ioutil"
	"strings"

	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsviz"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/repo"
)

func init() {
	stylesheet := `<style type="text/css">
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
</style>`

	header := `<header>
  <div class="content">
    <label style="color: white">Dataset</label>
    <h4 class="ref">{{ ds.peername}}/{{ ds.name }}</h4>
    {{ if ds.meta }}<h1>{{ ds.meta.title }}</h1>{{ end }}
  </div>
</header>`

	summary := `<section class="content">
  {{ if ds.meta -}}
  <p>{{ ds.meta.description }}</p>
  {{ end }}
  {{ if ds.structure }}
  <p class="stat"><label>data format:</label>{{ ds.structure.format }}</p>
  <p class="stat"><label>entry count:</label>{{ ds.structure.entries }}</p>
  <p class="stat"><label>errors:</label>{{ ds.structure.errCount }}</p>
  {{ end }}
</section>`

	citation := `<footer>
  <div class="content">
    <p class="stat"><label>commit title:</label>{{ ds.commit.title }}</p>
    <small>{{ ds.commit.timestamp }}</small><br />
    <small class="path">{{ ds.path }}</small><br />
    {{ if ds.meta.license }}
        <p>License: <a href="{{ ds.meta.license.url }}">{{ ds.meta.license.type }}</a></p>
    {{ end }}
  </div>
</footer>`

	dsviz.PredefinedHTMLTemplates = map[string]string{
		"stylesheet": stylesheet,
		"header":     header,
		"summary":    summary,
		"citation":   citation,
	}
}

// AddDefaultViz sets a dataset viz component & scriptFile if one isn't
// specified
func AddDefaultViz(ds *dataset.Dataset) {
	if ds.Viz == nil {
		ds.Viz = &dataset.Viz{Format: "html"}
	}

	if ds.Viz.ScriptFile() == nil {
		ds.Viz.SetScriptFile(qfs.NewMemfileReader("viz.html", strings.NewReader(DefaultTemplate)))
	}
}

// Render executes a template for a dataset, returning a slice of HTML
func Render(r repo.Repo, ref repo.DatasetRef, tmplData []byte) ([]byte, error) {
	const tmplName = "template"

	store := r.Store()

	ds, err := dsfs.LoadDataset(store, ref.Path)
	if err != nil {
		log.Debug(err.Error())
		return nil, err
	}
	if err := OpenDataset(r.Filesystem(), ds); err != nil {
		return nil, err
	}

	// TODO (b5): plzzzzzzz standardize this into one place
	ds.Peername = ref.Peername
	ds.Name = ref.Name

	AddDefaultViz(ds)

	if tmplData != nil {
		ds.Viz.SetScriptFile(qfs.NewMemfileBytes(tmplName, tmplData))
	}

	data, err := dsviz.Render(ds)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(data)
}

// DefaultTemplate is the template that render will fall back to should no
// template be available
var DefaultTemplate = `<!DOCTYPE html>
<html>
<head>
  <title>{{ title }}</title>
  {{ block "stylesheet" . }}{{ end }}
</head>
<body class="viewport">
  {{ block "header" . }}{{ end }}
  {{ block "summary" . }}{{ end }}
  {{ block "citation" . }}{{ end }}
</body>
</html>`
