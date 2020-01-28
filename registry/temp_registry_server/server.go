// Package regserver is a wrapper around the handlers package,
// turning it into a proper http server
package main

import (
	"context"
	"net/http"
	"os"

	logger "github.com/ipfs/go-log"
	"github.com/qri-io/qri/registry/regserver/handlers"
)

var (
	log      = logger.Logger("regserver")
	adminKey string
)

func main() {
	logger.SetLogLevel("regserver", "info")
	ctx := context.Background()
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	inst, reg, cleanup, err := NewTempRepoRegistry(ctx)
	if err != nil {
		log.Fatalf("creating temp registry: %s", err)
	}
	defer cleanup()

	addBasicDataset(inst)

	s := http.Server{
		Addr:    ":" + port,
		Handler: handlers.NewRoutes(reg),
	}

	log.Infof("serving on: %s", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Info(err.Error())
	}
}
