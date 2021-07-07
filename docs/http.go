package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func serveDocs(ctx context.Context, addr string) error {
	fmt.Println("serving api docs")
	var specBuf *bytes.Buffer
	specBuf, err := OpenAPIYAML()
	if err != nil {
		return err
	}

	go watchSpecFiles(ctx, specBuf, &err)

	s := &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/qri_api.yaml":
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
				}
				w.Write(specBuf.Bytes())
			default:
				w.Write([]byte(redocTemplate))
			}
		}),
	}

	return s.ListenAndServe()
}

const redocTemplate = `<!DOCTYPE html>
<html>
  <head>
    <title>ReDoc</title>
    <!-- needed for adaptive design -->
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <redoc spec-url='./qri_api.yaml'></redoc>
    <script src="https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js"> </script>
  </body>
</html>`

func watchSpecFiles(ctx context.Context, specBuf *bytes.Buffer, yamlErr *error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					buf, yamlE := OpenAPIYAML()
					*specBuf = *buf
					*yamlErr = yamlE
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			case <-ctx.Done():
				done <- true
			}
		}
	}()

	// add doc template
	err = watcher.Add("./api_doc_template.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// watch everything in the lib directory
	files, err := ioutil.ReadDir("../lib")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		path := filepath.Join("../lib", f.Name())
		watcher.Add(path)
	}

	<-done
}
