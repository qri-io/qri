# [qri](http://qri.io)

[![Qri](https://img.shields.io/badge/made%20by-qri-magenta.svg?style=flat-square)](https://qri.io)
[![GoDoc](https://godoc.org/github.com/qri-io/qri?status.svg)](http://godoc.org/github.com/qri-io/qri)
[![License](https://img.shields.io/github/license/qri-io/qri.svg?style=flat-square)](./LICENSE)
[![Codecov](https://img.shields.io/codecov/c/github/qri-io/qri.svg?style=flat-square)](https://codecov.io/gh/qri-io/qri)
[![CI](https://img.shields.io/circleci/project/github/qri-io/qri.svg?style=flat-square)](https://circleci.com/gh/qri-io/qri)

#### qri is a global dataset version control system (GDVCS) built on the distributed web

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

<p align="center">
  <a href="https://asciinema.org/a/160357" target="_blank"><img src="https://asciinema.org/a/160357.png" width="654"/></a>
</p>


## Getting Involved

We would love involvement from more people! If you notice any errors or would
like to submit changes, please see our
[Contributing Guidelines](./.github/CONTRIBUTING.md).

## Building From Source

To build qri you'll need the [go programming language](https://golang.org) on your machine. You need to have at least version 1.7, and can check your version of go by running:
```shell
$ go verison
```

You will also need to have go binaries on your path. By this, we mean the folder `$GOPATH/bin` should be in a place that your computer checks for binary executables. There are a few ways to set up your GOPATH, but we recommend checking out the [goLang wiki](https://github.com/golang/go/wiki/SettingGOPATH).

Once your GOPATH is set, run:

```shell
$ go get github.com/qri-io/qri
```

Ignore any errors about `unrecognized import path`, those are expected.

Alternatively, if you opt to clone the repository using `git`, be sure to clone it to a directory based on your GOPATH.

```shell
$ mkdir -p $GOPATH/src/github.com/qri-io
$ cd $GOPATH/src/github.com/qri-io
$ git clone git@github.com:qri-io/qri.git
```

Note - if you run into this error:

```shell
$ xcrun: error: invalid active developer path (/Library/Developer/CommandLineTools)
$ missing xcrun at: /Library/Developer/CommandLineTools/usr/bin/xcrun
```

You need to install XCode by running:
```shell
$ xcode-select --install
```

Once `go get` or `git clone` has finished, run:

```shell
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

## Developing

We've set up a separate document for [developer guidelines](https://github.com/qri-io/qri/blob/master/DEVELOPERS.md)!

## Troubleshooting

Please see our [troubleshooting document](https://github.com/qri-io/qri/blob/master/troubleshooting.md) for fixes to common issues setting up and running qri.
