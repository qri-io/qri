package lib

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/rpc"
	"strings"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/dataset/dsio"
	"github.com/qri-io/qri/repo"
)

// RenderRequests encapsulates business logic for this node's
// user profile
type RenderRequests struct {
	cli  *rpc.Client
	repo repo.Repo
}

// NewRenderRequests creates a RenderRequests pointer from either a repo
// or an rpc.Client
func NewRenderRequests(r repo.Repo, cli *rpc.Client) *RenderRequests {
	if r != nil && cli != nil {
		panic(fmt.Errorf("both repo and client supplied to NewRenderRequests"))
	}

	return &RenderRequests{
		cli:  cli,
		repo: r,
	}
}

// CoreRequestsName implements the Requets interface
func (RenderRequests) CoreRequestsName() string { return "render" }

// RenderParams defines parameters for the Render method
type RenderParams struct {
	Ref            repo.DatasetRef
	Template       []byte
	TemplateFormat string
	All            bool
	Limit, Offset  int
}

// Render executes a template against a template
func (r *RenderRequests) Render(p *RenderParams, res *[]byte) error {
	const tmplName = "template"
	var rdr io.Reader

	if r.cli != nil {
		return r.cli.Call("RenderRequests.Render", p, res)
	}

	if err := DefaultSelectedRef(r.repo, &p.Ref); err != nil {
		return err
	}

	err := repo.CanonicalizeDatasetRef(r.repo, &p.Ref)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	ref, err := r.repo.GetRef(p.Ref)
	if err != nil {
		log.Debug(err.Error())
		return NewError(err, fmt.Sprintf("could not find dataset '%s/%s'", p.Ref.Peername, p.Ref.Name))
	}

	store := r.repo.Store()

	ds, err := dsfs.LoadDataset(store, datastore.NewKey(ref.Path))
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	if p.Template != nil {
		rdr = bytes.NewBuffer(p.Template)
	}

	// TODO - hack for now. a subpackage of dataset should handle all of the below,
	// and use a method to set the default template if one can be loaded from the web
	if rdr == nil && ds.Viz != nil && ds.Viz.ScriptPath != "" {
		f, err := store.Get(datastore.NewKey(ds.Viz.ScriptPath))
		if err != nil {
			return fmt.Errorf("loading template from store: %s", err.Error())
		}
		rdr = f
	}

	if rdr == nil {
		rdr = strings.NewReader(DefaultTemplate)
	}

	tmplBytes, err := ioutil.ReadAll(rdr)
	if err != nil {
		return fmt.Errorf("reading template data: %s", err.Error())
	}

	tmpl, err := template.New(tmplName).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("parsing template: %s", err.Error())
	}

	file, err := dsfs.LoadBody(store, ds)
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	var (
		array []interface{}
		obj   = map[string]interface{}{}
		read  = 0
	)

	tlt := ds.Structure.Schema.TopLevelType()

	rr, err := dsio.NewEntryReader(ds.Structure, file)
	if err != nil {
		return fmt.Errorf("error allocating data reader: %s", err)
	}

	for i := 0; i >= 0; i++ {
		val, err := rr.ReadEntry()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("row iteration error: %s", err.Error())
		}
		if !p.All && i < p.Offset {
			continue
		}

		if tlt == "object" {
			obj[val.Key] = val.Value
		} else {
			array = append(array, val.Value)
		}

		read++
		if read == p.Limit {
			break
		}
	}

	enc := ds.Encode()
	// TODO - repo.DatasetRef should be refactored into this newly expanded DatasetPod,
	// once that's done these values should be populated by ds.Encode(), removing the need
	// for these assignments
	enc.Peername = ref.Peername
	enc.ProfileID = ref.ProfileID.String()
	enc.Name = ref.Name
	if enc.Meta == nil {
		enc.Meta = &dataset.Meta{}
	}

	if tlt == "object" {
		enc.Body = obj
	} else {
		enc.Body = array
	}

	tmplBuf := &bytes.Buffer{}
	if err := tmpl.Execute(tmplBuf, enc); err != nil {
		return err
	}

	*res = tmplBuf.Bytes()
	return nil
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
