package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/qri-io/qfs/cafs"
)

// ServeWebapp launches a webapp server on s.cfg.Webapp.Port
func (s Server) ServeWebapp(ctx context.Context) {
	cfg := s.Config()
	if !cfg.Webapp.Enabled || cfg.Webapp.Port == 0 {
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Webapp.Port))
	if err != nil {
		log.Infof("Webapp listen on port %d error: %s", cfg.Webapp.Port, err)
		return
	}

	m := webappMuxer{
		template: s.middleware(s.WebappTemplateHandler),
		webapp:   s.FrontendHandler(ctx, "/webapp"),
	}

	webappserver := &http.Server{Handler: m}

	go func() {
		<-ctx.Done()
		log.Info("closing webapp server")
		webappserver.Close()
	}()

	webappserver.Serve(listener)
	return
}

type webappMuxer struct {
	webapp, template http.Handler
}

func (h webappMuxer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/webapp") {
		h.webapp.ServeHTTP(w, r)
		return
	}

	h.template.ServeHTTP(w, r)
}

// FrontendHandler fetches the compiled frontend webapp using its hash
// and serves it up as a traditional HTTP endpoint, transparently redirecting requests
// for [prefix]/foo.js to [CAFS Hash]/foo.js
// prefix is the path prefix that should be stripped from the request URL.Path
func (s Server) FrontendHandler(ctx context.Context, prefix string) http.Handler {
	// TODO -
	// * there's no error handling,
	// * and really the data could be owned by Server and initialized by it,
	//  as there's nothing that necessitates updating the webapp within FrontendHandler.
	updating := true
	var fetchErr error
	errs := make(chan error)
	go func() {
		if err := s.resolveWebappPath(); err != nil {
			errs <- err
			return
		}
		path := s.Config().Webapp.EntrypointHash

		log.Info("fetching webapp off the distributed web...")
		fetcher, ok := s.Repo().Store().(cafs.Fetcher)
		if !ok {
			errs <- fmt.Errorf("this store cannot fetch from remote sources")
			return
		}
		_, err := fetcher.Fetch(ctx, cafs.SourceAny, path+"/main.js")
		if err != nil {
			errs <- fmt.Errorf("error fetching file: %s", err.Error())
		}
		log.Info("pinning webapp...")
		pinner, ok := s.Repo().Store().(cafs.Pinner)
		if !ok {
			errs <- fmt.Errorf("this store is not configured to pin")
			return
		}
		if err := pinner.Pin(ctx, path, true); err != nil {
			errs <- fmt.Errorf("error pinning webapp: %s", err.Error())
			return
		}
		log.Info("done pinning webapp!")
		errs <- nil
	}()

	go func() {
		select {
		case fetchErr = <-errs:
			if fetchErr != nil {
				log.Errorf("error fetching webapp: %s", fetchErr)
			}
			updating = false
		case <-ctx.Done():
			return
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if updating == true {
			// if the app is still pinning
			// return a temporary script that automatically refreshes and tells the user what's going on
			w.Write([]byte(loadingScript))
			return
		}

		if fetchErr != nil {
			errScript := strings.Replace(errScriptTemplate, "ERROR", fetchErr.Error(), 1)
			w.Write([]byte(errScript))
			return
		}
		path := fmt.Sprintf("%s%s", s.Config().Webapp.EntrypointHash, strings.TrimPrefix(r.URL.Path, prefix))
		s.fetchCAFSPath(path, w, r)
	})
}

// resolveWebappPath resolved the current webapp hash
func (s Server) resolveWebappPath() error {
	node := s.Node()
	cfg := s.Config()
	if cfg.Webapp.EntrypointUpdateAddress == "" {
		log.Debug("no entrypoint update address specified for update checking")
		return nil
	}

	namesys, err := node.GetIPFSNamesys()
	if err != nil {
		log.Debugf("no IPFS node present to resolve webapp address: %s", err.Error())
		return nil
	}

	p, err := namesys.Resolve(context.Background(), cfg.Webapp.EntrypointUpdateAddress)
	if err != nil {
		return fmt.Errorf("error resolving IPNS Name: %s", err.Error())
	}
	updatedPath := p.String()
	if updatedPath != cfg.Webapp.EntrypointHash {
		log.Infof("updated webapp path to version: %s", updatedPath)
		cfg.Set("webapp.entrypointhash", updatedPath)
		if err := s.ChangeConfig(cfg); err != nil {
			return fmt.Errorf("error updating config: %s", err)
		}
	}
	return nil
}

// WebappTemplateHandler renders the home page
func (s Server) WebappTemplateHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(s.Config().Webapp, w, "webapp")
}

const loadingScript = `
var css = document.createElement("style");
css.type = 'text/css';

var styles = "@keyframes spinnerAnim { 0% { transform: scaleY(0.4); } 20% { transform: scaleY(1.0); } 40% { transform: scaleY(0.4); } 100% { transform: scaleY(0.4);}}"
styles += " .spinner-spinner { width: 60px; height: 40px; text-align: center; font-size: 10px; display: inline-block;}"
styles += " .spinner-block { height: 100%; width: 6px; margin-right: 2px; border-radius: 6px; display: inline-block; animation-name: spinnerAnim; animation-duration: 1.2s; animation-iteration-count: infinite; background: #303030;}"
styles += " .spinner-rect1 { animation-delay: 0s;}"
styles += " .spinner-rect2 { animation-delay: -1.1s;}"
styles += " .spinner-rect3 { animation-delay: -1s;}"
styles += " .spinner-rect4 { animation-delay: -1s;}"
styles += " .spinner-rect5 { animation-delay: -0.9s;}"
styles += " .wrapper {	background: #FFF;	position: absolute;	width: 100%;	height: 100%;	top: 0	left: 0;}"
styles += " .title {	font-size: 18px;	line-height: 30px;	font-family: 'Rubik', 'Avenir Next', Arial, sans-serif;}"
styles += " p { font-size: 14px; line-height: 18px; font-family: 'Rubik', 'Avenir Next', Arial, sans-serif; font-weight: 300;}"
styles += " .center {	width: 500px;	height: 300px;	margin: 10em auto;	top: 0;	left: 0;	bottom: 0;	right: 0;	text-align: center;}"

if (css.styleSheet) css.styleSheet.cssText = styles;
else css.appendChild(document.createTextNode(styles));

var head = document.getElementsByTagName('head')[0].appendChild(css);

var spinner = document.createElement("div");
spinner.classList.add("spinner-spinner");

var rect1 = document.createElement("div");
var rect2 = document.createElement("div");
var rect3 = document.createElement("div");
var rect4 = document.createElement("div");
var rect5 = document.createElement("div");

rect1.classList.add("spinner-block", "spinner-rect1", "spinner-dark");
rect2.classList.add("spinner-block", "spinner-rect2", "spinner-dark");
rect3.classList.add("spinner-block", "spinner-rect3", "spinner-dark");
rect4.classList.add("spinner-block", "spinner-rect4", "spinner-dark");
rect5.classList.add("spinner-block", "spinner-rect5", "spinner-dark");

spinner.appendChild(rect1);
spinner.appendChild(rect2);
spinner.appendChild(rect3);
spinner.appendChild(rect4);
spinner.appendChild(rect5);

var center = document.createElement("div");
center.classList.add("center")
var title = document.createElement("div")
title.classList.add("title")
title.innerHTML = "downloading the Qri webapp from the distributed web!";
var p = document.createElement("p");
p.innerHTML = "if you are not automatically redirected in a few seconds please refresh"
center.appendChild(title)
center.appendChild(p)
center.appendChild(spinner)

var root = document.getElementById("root");
root.classList.add("wrapper")
root.appendChild(center);

setTimeout(function(){
   window.location.reload(1);
}, 1000);
`

const errScriptTemplate = `
var css = document.createElement("style");
css.type = 'text/css';

var styles = " .wrapper {	background: #FFF;	position: absolute;	width: 100%;	height: 100%;	top: 0	left: 0;}"
styles += " .title {	font-size: 18px;	line-height: 30px;	font-family: 'Rubik', 'Avenir Next', Arial, sans-serif;}"
styles += " p { font-size: 14px; line-height: 18px; font-family: 'Rubik', 'Avenir Next', Arial, sans-serif; font-weight: 300;}"
styles += " .center {	width: 500px;	height: 300px;	margin: 10em auto;	top: 0;	left: 0;	bottom: 0;	right: 0;	text-align: center;}"

if (css.styleSheet) css.styleSheet.cssText = styles;
else css.appendChild(document.createTextNode(styles));

var head = document.getElementsByTagName('head')[0].appendChild(css);


var center = document.createElement("div");
center.classList.add("center")
var title = document.createElement("div")
title.classList.add("title")
title.innerHTML = "There was an error downloading the Qri webapp. </br>Restart Qri to try again.";
var p = document.createElement("p");
p.innerHTML = "ERROR"
center.appendChild(title)
center.appendChild(p)

var root = document.getElementById("root");
root.classList.add("wrapper")
root.appendChild(center);
`
