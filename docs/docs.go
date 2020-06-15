// Docs is a tool for generating markdown documentation of the qri command line interface (CLI)
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
	qri "github.com/qri-io/qri/cmd"
	"github.com/qri-io/qri/repo/gen"
	"github.com/spf13/cobra/doc"
)

func main() {
	var (
		dir        string
		merge      string
		filenames  []string
		aggregator []byte
	)

	flag.StringVar(&dir, "dir", "docs", "path to the docs directory")
	flag.StringVar(&merge, "filename", "", "docs will be merged into one markdown file with this filename. default extension: markdown")

	flag.Parse()

	ctx := context.Background()

	// generate markdown filenames
	root, _ := qri.NewQriCommand(ctx, qri.EnvPathFactory, gen.NewCryptoSource(), ioes.NewStdIOStreams())

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
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
|[qri add](#qri_add)  | Add a dataset from another peer   |
|[qri body](#qri_body)  | Get the body of a dataset   |
|[qri config](#qri_config)  | Get and set local configuration information   |
|[qri connect](#qri_connect)  | Connect to the distributed web by spinning up a Qri node   |
|[qri diff](#qri_diff)  | Compare differences between two datasets   |
|[qri export](#qri_export)  | Copy datasets to your local filenamesystem   |
|[qri get](#qri_get)  | Get elements of qri datasets   |
|[qri info](#qri_info)  | Show summarized description of a dataset   |
|[qri list](#qri_list)  | Show a list of datasets   |
|[qri log](#qri_log)  | Show log of dataset history   |
|[qri new](#qri_new)  | Create a new dataset   |
|[qri peers](#qri_peers)  | Commands for working with peers   |
|[qri registry](#qri_registry)  | Commands for working with a qri registry   |
|[qri remove](#qri_remove)  | Remove a dataset from your local repository   |
|[qri rename](#qri_rename)  | Change the name of a dataset   |
|[qri render](#qri_render)  | Execute a template against a dataset   |
|[qri save](#qri_save)  | Save changes to a dataset   |
|[qri search](#qri_search)  | Search qri   |
|[qri setup](#qri_setup)  | Initialize qri and IPFS repositories, provision a new qri ID   |
|[qri use](#qri_use)  | Select datasets for use with the qri get command   |
|[qri validate](#qri_validate)  | Show schema validation errors   |
|[qri version](#qri_version)  | Print the version number   |

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
