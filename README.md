# [qri](http://qri.io)

[![Qri](https://img.shields.io/badge/made%20by-qri-magenta.svg?style=flat-square)](https://qri.io) [![GoDoc](https://godoc.org/github.com/qri-io/qri?status.svg)](http://godoc.org/github.com/qri-io/qri) [![License](https://img.shields.io/github/license/qri-io/qri.svg?style=flat-square)](./LICENSE) [![Codecov](https://img.shields.io/codecov/c/github/qri-io/qri.svg?style=flat-square)](https://codecov.io/gh/qri-io/qri) [![CI](https://img.shields.io/circleci/project/github/qri-io/qri.svg?style=flat-square)](https://circleci.com/gh/qri-io/qri)

<h1 align="center">Qri CLI</h1>

<div align="center">
  <img alt="logo" src="https://qri.io/img/blobs/blob_trio.png" width="128">
</div>
<div align="center">
  <strong>a dataset version control system built on the distributed web</strong>
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
    <a href="https://github.com/qri-io/qri/blob/master/CONTRIBUTOR.md">
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
| "I want to download Qri" | [Download Qri](https://qri.io/download) or `brew install qri-io/qri/qri` |
| "I have a question" | [Create an issue](https://github.com/qri-io/qri/issues) and use the label 'question' |
| "I found a bug" | [Create an issue](https://github.com/qri-io/qri/issues) and use the label 'bug' |
| "I want to help build the Qri backend" | [Read the Contributing guides](https://github.com/qri-io/qri/blob/master/CONTRIBUTOR.md) |
| "I want to build Qri from source" | [Build Qri from source](#build)

__qri is a global dataset version control system built on the distributed web__

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

<a id="build"></a>
## Building From Source

To build qri you'll need the [go programming language](https://golang.org/dl/) on your machine.

```shell
$ git clone https://github.com/qri-io/qri
$ cd qri
$ make install
```

If this is your first time building, this command will have a lot of output. That's good! Its means it's working :) It'll take a minute or two to build.

After this is done, there will be a new binary `qri` in your `~/go/bin` directory if using go modules, and `$GOPATH/bin` directory otherwise. You should be able to run:

```shell
$ qri help
```
and see help output.

### Building on Windows

To start, make sure that you have enabled [Developer Mode](https://docs.microsoft.com/en-us/windows/uwp/get-started/enable-your-device-for-development). A library that we depend on needs it enabled in order to properly handle symlinks. If not done, you'll likely get the error message "A required privilege is not held by the client".

You should not need to Run As Administrator to build or run `qri`. We do not recommend using administrator to run `qri`.

#### Shell

For your shell, we recommend using [msys2](https://www.msys2.org/). Other shells, such as `cmd`, `Powershell`, or `cygwin` may also be usable, but `msys2` makes it easy to install our required dependencies. IPFS also recommends `msys2`, and `qri` is built on top of IPFS.

#### Dependencies

Building depends upon having `git` and `make` installed. If using `msys2`, you can easily install these by using the package manager "pacman". In a shell, type:

```shell
pacman -S git make
```

Assuming you've also installed `go` using the official Windows installer linked above, you will also need to add `go` to your `PATH` by modifying your environment variable. See the next section on "Environment variables" for more information.

Due to how msys2 treats the `PATH` variable, you also need to add a new environment variable `MSYS2_PATH_TYPE`, with the value `inherit`, using the same procedure.

Once these steps are complete, proceed to <a href="#building">building</a>.

### Building on Rasberry PI

On a Raspberry PI, you'll need to increase your swap file size in order to build. Normal desktop and server linux OSes should be fine to proceed to <a href="#building">building</a>.

One symptom of having not enough swap space is the `go install` command producing an error message ending with:

```
link: signal: killed
```

To increase your swapfile size, first turn off the swapfile:

```
sudo dphys-swapfile swapoff
```

Then edit `/etc/dphys-swapfile` as root and set `CONF_SWAPSIZE` to 1024.

Finally turn on the swapfile again:

```
sudo dphys-swapfile swapon
```

Otherwise linux machines with reduced memory will have other ways to increase their swap file sizes. Check documentation for your particular machine.


## Packages

Qri is comprised of many specialized packages. Below you will find a summary of each package.

| Package | Go Docs | Go Report Card | Description |
|---------|---------|----------------|-------------|
| [`api`](https://github.com/qri-io/qri/tree/master/api) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/api) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | user accessible layer, primarily made for communication with our frontend webapp |
| [`cmd`](https://github.com/qri-io/qri/tree/master/cmd) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/cmd) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | our command line interface |
| [`config`](https://github.com/qri-io/qri/tree/master/config) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/config) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | user configuration details, includes peer's profile |
| [`lib`](https://github.com/qri-io/qri/tree/master/lib) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/lib) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | takes arguments from the cmd and api layer and forms proper requests to call to the action layer |
| [`p2p`](https://github.com/qri-io/qri/tree/master/p2p) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/p2p) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | the peer to peer communication layer of qri |
| [`repo`](https://github.com/qri-io/qri/tree/master/repo) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/repo) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | the repository: saving, removing, and storing datasets, profiles, and the config |
| [`dataset`](https://github.com/qri-io/dataset) | [![Go Docs](https://godoc.org/github.com/qri-io/dataset?status.svg)](https://godoc.org/github.com/qri-io/dataset) | [![report](https://goreportcard.com/badge/github.com/qri-io/dataset)](https://goreportcard.com/report/github.com/qri-io/dataset) | the blueprint for a dataset, the atoms that make up qri |
| [`registry`](https://github.com/qri-io/qri/tree/master/registry) | [![Go Docs](https://godoc.org/github.com/qri-io/qri?status.svg)](https://godoc.org/github.com/qri-io/qri/registry) | [![report](https://goreportcard.com/badge/github.com/qri-io/qri)](https://goreportcard.com/report/github.com/qri-io/qri) | the blueprint for a registry: the service that allows profiles to be unique and datasets to be searchable |
| [`starlib`](https://github.com/qri-io/starlib) | [![Go Docs](https://godoc.org/github.com/qri-io/starlib?status.svg)](https://godoc.org/github.com/qri-io/starlib) | [![report](https://goreportcard.com/badge/github.com/qri-io/starlib)](https://goreportcard.com/report/github.com/qri-io/starlib) | the starlark standard library available for qri transform scripts |
| [`qfs`](https://github.com/qri-io/qfs) | [![Go Docs](https://godoc.org/github.com/qri-io/qfs?status.svg)](https://godoc.org/github.com/qri-io/qfs) | [![report](https://goreportcard.com/badge/github.com/qri-io/qfs)](https://goreportcard.com/report/github.com/qri-io/qfs) | "qri file sytem" is Qri's file system abstraction for getting & storing data from different sources |
| [`ioes`](https://github.com/qri-io/ioes) | [![Go Docs](https://godoc.org/github.com/qri-io/ioes?status.svg)](https://godoc.org/github.com/qri-io/ioes) | [![report](https://goreportcard.com/badge/github.com/qri-io/ioes)](https://goreportcard.com/report/github.com/qri-io/ioes) | package to handle in, out, and error streams: gives us better control of where we send output and errors |
| [`jsonschema`](https://github.com/qri-io/jsonschema) | [![Go Docs](https://godoc.org/github.com/qri-io/jsonschema?status.svg)](https://godoc.org/github.com/qri-io/jsonschema) | [![report](https://goreportcard.com/badge/github.com/qri-io/jsonschema)](https://goreportcard.com/report/github.com/qri-io/jsonschema) | used to describe the structure of a dataset, so we can validate datasets and determine dataset interop |

### Outside Libraries

The following packages are not under Qri, but are important dependencies, so we display their latest versions for convenience.

| Package | Version |
|--------|-------|
| `ipfs` | [![ipfs version](https://img.shields.io/badge/ipfs-v0.6.0-blue.svg)](https://github.com/ipfs/go-ipfs/) |


###### This documentation has been adapted from the [Cycle.js](https://github.com/cyclejs/cyclejs) documentation.
