# [qri](http://qri.io)

[![Qri](https://img.shields.io/badge/made%20by-qri-magenta.svg?style=flat-square)](https://qri.io) [![GoDoc](https://godoc.org/github.com/qri-io/qri?status.svg)](http://godoc.org/github.com/qri-io/qri) [![License](https://img.shields.io/github/license/qri-io/qri.svg?style=flat-square)](./LICENSE) [![Codecov](https://img.shields.io/codecov/c/github/qri-io/qri.svg?style=flat-square)](https://codecov.io/gh/qri-io/qri) [![CI](https://img.shields.io/circleci/project/github/qri-io/qri.svg?style=flat-square)](https://circleci.com/gh/qri-io/qri)

<h1 align="center">Qri Backend and CLI</h1>

<div align="center">
  <img alt="logo" src="https://qri.io/img/blobs/blob_trio.png" width="128">
</div>
<div align="center">
  <strong>a global dataset version control system (GDVCS) built on the distributed web</strong>
</div>

<div align="center">
  <h3>
    <a href="https://qri.io">
      Website
    </a>
    <span> | </span>
    <a href="#packages">
      Packages
    </a>
    <span> | </span>
    <a href="https://github.com/qri-io/qri/CONTRIBUTOR.md">
      Contribute
    </a>
    <span> | </span>
    <a href="https://github.com/qri-io/qri/issues">
      Issues
    </a>
     <span> | </span>
    <a href="https://qri.io/docs/">
      Docs
    </a>
     <span> | </span>
    <a href="https://qri.io/download/">
      Download
    </a>
  </h3>
</div>

<div align="center">
  <!-- Build Status -->
</div>

## Welcome

| Question | Answer |
|--------|-------|
| "I want to learn about Qri" | [Read the official documentation](https://qri.io/docs/) |
| "I want to download Qri" | [Download Qri](https://qri.io/download) |
| "I have a question" | [Create an issue](https://github.com/qri-io/qri/issues) and use the label 'question' |
| "I found a bug" | [Create an issue](https://github.com/qri-io/qri/issues) and use the label 'bug' |
| "I want to help build the Qri backend" | [Read the Contributing guides](https://github.com/qri-io/qri/CONTRIBUTOR.md) |
| "I want to build Qri from source" | [Build Qri from source](#build)

__qri is a global dataset version control system (GDVCS) built on the distributed web__

Breaking that down:

- **global** so that if *anyone, anywhere* has published work with the same or similar datasets, you can discover it.
- Specific to **datasets** because data deserves purpose-built tools
- **version control** to keep data in sync, attributing all changes to authors
- On the **distributed web** to make *all* of the data published on qri simultaneously available, letting peers work on data together.
 
If you’re unfamiliar with *version control,* particularly the distributed kind, well you're probably viewing this document on [github](https://github.com/qri-io/qri) — which is a version control system intended for code. Its underlying technology – git – popularized some magic sauce that has inspired a generation of programmers and popularized concepts at the heart of the distributed web. Qri is applying that family of concepts to four common data problems:

1. **Discovery** _Can I find data I’m looking for?_
2. **Trust** _Can I trust what I’ve found?_
3. **Friction** _Can I make this work with my other stuff?_
4. **Sync** _How do I handle changes in data?_

Because qri is *global* and *content-addressed*, adding data to qri also checks the entire network to see if someone has added it before. Since qri is focused solely on datasets, it can provide meaningful search results. Every change on qri is associated with a peer, creating an audit-able trail you can use to quickly see what has changed and who has changed it. All datasets on qri are automatically described at the time of ingest using a flexible schema that makes data naturally inter-operate. Qri comes with tools to turn *all* datasets on the network into a JSON API with a single command. Finally, all changes in qri are tracked & synced.

## Packages

Qri is comprised of many specialized packages. Below you will find a summary of each package.

| Package | Go Docs | Go Report Card | Description |
|---------|---------|----------------|-------------|
| [`@qri/actions`](https://github.com/qri-io/qri/tree/master/actions) | <img width=190/>[![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/actions) | <img width=165/>[![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | functions that call to the repo to carry out tasks  |
| [`@qri/api`](https://github.com/qri-io/qri/tree/master/api) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/api) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | user accessible layer, primarily made for communication with our frontend webapp |
| [`@qri/cmd`](https://github.com/qri-io/qri/tree/master/cmd) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/cmd) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | our command line interface |
| [`@qri/config`](https://github.com/qri-io/qri/tree/master/config) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/config) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | user configuration details, includes peer's profile |
| [`@qri/lib`](https://github.com/qri-io/qri/tree/master/lib) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/lib) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | takes arguments from the cmd and api layer and forms proper requests to call to the action layer |
| [`@qri/p2p`](https://github.com/qri-io/qri/tree/master/p2p) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/p2p) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | the peer to peer communication layer of qri |
| [`@qri/repo`](https://github.com/qri-io/qri/tree/master/repo) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/repo) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | the repository: saving, removing, and storing datasets, profiles, and the config |
| [`@dataset`](https://github.com/qri-io/dataset) | [![Go Docs](https://godoc.org/github.com/qri-io/dataset?status.svg)](https://godoc.org/github.com/qri-io/dataset) | [![report](https://goreportcard.com/badge/github.com/qri-io/dataset)](https://goreportcard.com/report/github.com/qri-io/dataset) | the blueprint for a dataset, the atoms that make up qri |
| [`@registry`](https://github.com/qri-io/registry) | [![Go Docs](https://godoc.org/github.com/qri-io/registry?status.svg)](https://godoc.org/github.com/qri-io/registry) | [![report](https://goreportcard.com/badge/github.com/qri-io/registry)](https://goreportcard.com/report/github.com/qri-io/registry) | the blueprint for a registry: the service that allows profiles to be unique and datasets to be searchable |
| [`@skytf`](https://github.com/qri-io/skytf) | [![Go Docs](https://godoc.org/github.com/qri-io/skytf?status.svg)](https://godoc.org/github.com/qri-io/skytf) | [![report](https://goreportcard.com/badge/github.com/qri-io/skytf)](https://goreportcard.com/report/github.com/qri-io/skytf) | brings starlark into qri to be used in transforms, adds qri specific functionality |
| [`@starlib`](https://github.com/qri-io/starlib) | [![Go Docs](https://godoc.org/github.com/qri-io/starlib?status.svg)](https://godoc.org/github.com/qri-io/starlib) | [![report](https://goreportcard.com/badge/github.com/qri-io/starlib)](https://goreportcard.com/report/github.com/qri-io/starlib) | the starlark standard library available for qri transform scripts |
| [`@cafs`](https://github.com/qri-io/cafs) | [![Go Docs](https://godoc.org/github.com/qri-io/cafs?status.svg)](https://godoc.org/github.com/qri-io/cafs) | [![report](https://goreportcard.com/badge/github.com/qri-io/cafs)](https://goreportcard.com/report/github.com/qri-io/cafs) | stands for "content addressed file system", this is the seam that communicates with a specific CAFS, current options are IPFS and an in-memory file system |
| [`@ioes`](https://github.com/qri-io/ioes) | [![Go Docs](https://godoc.org/github.com/qri-io/ioes?status.svg)](https://godoc.org/github.com/qri-io/ioes) | [![report](https://goreportcard.com/badge/github.com/qri-io/ioes)](https://goreportcard.com/report/github.com/qri-io/ioes) | package to handle in, out, and error streams: gives us better control of where we send output and errors |
| [`@dsdiff`](https://github.com/qri-io/dsdiff) | [![Go Docs](https://godoc.org/github.com/qri-io/dsdiff?status.svg)](https://godoc.org/github.com/qri-io/dsdiff) | [![report](https://goreportcard.com/badge/github.com/qri-io/dsdiff)](https://goreportcard.com/report/github.com/qri-io/dsdiff) | the dataset diffing package |
| [`@jsonschema`](https://github.com/qri-io/jsonschema) | [![Go Docs](https://godoc.org/github.com/qri-io/jsonschema?status.svg)](https://godoc.org/github.com/qri-io/jsonschema) | [![report](https://goreportcard.com/badge/github.com/qri-io/jsonschema)](https://goreportcard.com/report/github.com/qri-io/jsonschema) | used to describe the structure of a dataset, so we can validate datasets and determine dataset interop |

### Outside Libraries

The following packages are not under Qri, but are important dependencies, so we display their latest versions for convenience.

| Package | Version |
|--------|-------|
| `ipfs` | [![ipfs version](https://img.shields.io/badge/ipfs-v0.4.17-blue.svg)](https://github.com/ipfs/go-ipfs/) |

<a id="build"></a>
### Building From Source

To build qri you'll need the [go programming language](https://golang.org) on your machine.

```shell
$ go get github.com/qri-io/qri
$ cd $GOPATH/src/github.com/qri-io/qri
$ make build
$ go install
```

The `make build` command will have a lot of output. That's good! Its means it's working :)

It'll take a minute, but once everything's finished a new binary `qri` will appear in the `$GOPATH/bin` directory. You should be able to run:
```shell
$ qri help
```
and see help output.


###### This documentation has been adapted from the [Cycle.js](https://github.com/cyclejs/cyclejs) documentation.