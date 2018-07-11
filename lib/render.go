package lib

import (
	"bytes"
	"fmt"
	"html/template"
	// "io"
	"io/ioutil"
	// "net/http"
	"net/rpc"
	// "strings"

	"github.com/ipfs/go-datastore"
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

	// if p.Template == nil && Config != nil && Config.Render != nil && Config.Render.DefaultTemplateHash != "" {
	// 	log.Debugf("using default hash: %s", Config.Render.DefaultTemplateHash)
	// 	var rdr io.Reader
	// 	file, err := store.Get(datastore.NewKey(Config.Render.DefaultTemplateHash))
	// 	if err != nil {
	// 		if strings.Contains(err.Error(), "not found") && Config.P2P != nil && Config.P2P.HTTPGatewayAddr != "" {
	// 			log.Debugf("fetching %d from ipfs gateway", Config.Render.DefaultTemplateHash)
	// 			var res *http.Response
	// 			res, err = http.Get(fmt.Sprintf("%s%s", Config.P2P.HTTPGatewayAddr, Config.Render.DefaultTemplateHash))
	// 			if err != nil {
	// 				return err
	// 			}
	// 			defer res.Body.Close()
	// 			rdr = res.Body
	// 		} else {
	// 			return fmt.Errorf("loading default template: %s", err.Error())
	// 		}
	// 	} else {
	// 		rdr = file
	// 	}

	// 	p.Template, err = ioutil.ReadAll(rdr)
	// 	if err != nil {
	// 		return fmt.Errorf("reading template: %s", err.Error())
	// 	}
	// }

	tmpl, err := template.New("template").Parse(string(p.Template))
	if err != nil {
		return fmt.Errorf("parsing template: %s", err.Error())
	}

	ds, err := dsfs.LoadDataset(store, datastore.NewKey(ref.Path))
	if err != nil {
		log.Debug(err.Error())
		return err
	}

	// TODO - hack for now
	if ds.Viz != nil && ds.Viz.ScriptPath != "" {
		f, err := store.Get(datastore.NewKey(ds.Viz.ScriptPath))
		if err != nil {
			return fmt.Errorf("loading template from store: %s", err.Error())
		}

		tmplBytes, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("reading template data: %s", err.Error())
		}

		tmpl, err = template.New("template").Parse(string(tmplBytes))
		if err != nil {
			return fmt.Errorf("parsing template: %s", err.Error())
		}
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
