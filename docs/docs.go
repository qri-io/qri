// Docs is a tool for generating markdown documentation of the qri command line
// interface (CLI)
package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/auth/key"
	qri "github.com/qri-io/qri/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	var (
		dir        string
		api        bool
		apiOnly    bool
		merge      string
		filenames  []string
		aggregator []byte
	)

	flag.StringVar(&dir, "dir", "docs", "path to the docs directory")
	flag.StringVar(&merge, "filename", "", "docs will be merged into one markdown file with this filename. default extension: markdown")
	flag.BoolVar(&api, "api", false, "docs will generate the api spec")
	flag.BoolVar(&apiOnly, "apiOnly", false, "docs will generate the api spec and stop")

	flag.Parse()
	api = api || apiOnly

	ctx := context.Background()

	// generate markdown filenames
	root, _ := qri.NewQriCommand(ctx, qri.StandardRepoPath(), key.NewCryptoSource(), ioes.NewStdIOStreams())

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}

	if api {
		buf, err := OpenAPIYAML()
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		err = ioutil.WriteFile(filepath.Join(dir, "open_api_spec.yaml"), buf.Bytes(), 0777)
		if err != nil {
			panic(err)
		}
		if apiOnly {
			os.Exit(0)
		}
	}

	fileList := func(name string) string {
		name = filepath.Base(name)
		filenames = append(filenames, name)
		return "<a id='" + name[:len(name)-len(filepath.Ext(name))] + "'></a>\n"
	}

	err := doc.GenMarkdownTreeCustom(root, dir, fileList, func(string) string { return "" })
	if err != nil {
		log.Fatal(err)
	}

	// if we are supplied with a filename, merge docs into one file and save with that filename
	if merge != "" {

		data := []byte(`---
title: "CLI commands"
description: "qri command line reference"
date: 2018-01-30T00:00:00-04:00
section: reference
draft: false
---

# Qri CLI command line reference 


Qri ("query") is a global dataset version control system 
on the distributed web.


|Command | Description |
|--------|-------------|
|[qri access](#qri_access_token)  | Create an access token   |
|[qri apply](#qri_apply)  | Apply a transform to a dataset   |
|[qri checkout](#qri_checkout)  | Create a directory linked to a dataset   |
|[qri completion](#qri_completion)  | Generate shell auto-completion scripts   |
|[qri config](#qri_config)  | Get and set local configuration information   |
|[qri connect](#qri_connect)  | Connect to the distributed web by spinning up a Qri node   |
|[qri diff](#qri_diff)  | Compare differences between two datasets   |
|[qri get](#qri_get)  | Get elements of qri datasets   |
|[qri init](#qri_init)  | Initialize a dataset directory   |
|[qri list](#qri_list)  | Show a list of datasets   |
|[qri log](#qri_log)  | Show log of dataset history   |
|[qri peers](#qri_peers)  | Commands for working with peers   |
|[qri preview](#qri_preview)  | Fetch a dataset preview   |
|[qri pull](#qri_pull)  | Fetch and store datasets from other peers   |
|[qri push](#qri_push)  | Send a dataset to a remote   |
|[qri registry](#qri_registry)  | Commands for working with a qri registry   |
|[qri remove](#qri_remove)  | Remove a dataset from your local repository   |
|[qri rename](#qri_rename)  | Change the name of a dataset   |
|[qri render](#qri_render)  | Execute a template against a dataset   |
|[qri restore](#qri_restore)  | Restore a checked out dataset's files to a previous state   |
|[qri save](#qri_save)  | Save changes to a dataset   |
|[qri search](#qri_search)  | Search qri   |
|[qri setup](#qri_setup)  | Initialize qri and IPFS repositories, provision a new qri ID   |
|[qri sql](#qri_sql)  | Experimental: Run an SQL query on local dataset(s)   |
|[qri status](#qri_status)  | Show what components of a dataset have been changed   |
|[qri use](#qri_use)  | Select datasets for use with the qri get command   |
|[qri validate](#qri_validate)  | Show schema validation errors   |
|[qri version](#qri_version)  | Print the version number   |
|[qri workdir](#qri_workdir)  | File system integration tools   |

________

`)

		aggregator = append(aggregator, data...)

		for _, file := range filenames {

			if file == "qri.md" {
				continue
			}

			data, err = ioutil.ReadFile(filepath.Join(dir, file))
			if err != nil {
				panic(err)
			}

			index := strings.Index(string(data), "### SEE ALSO")
			if index < 0 {
				index = len(data)
			}
			aggregator = append(aggregator, data[:index]...)
			aggregator = append(aggregator, []byte("\n\n________\n\n")...)
		}

		if filepath.Ext(merge) == "" {
			merge += ".md"
		}

		err = ioutil.WriteFile(filepath.Join(dir, merge), aggregator, 0777)
		if err != nil {
			panic(err)
		}

	}
}
