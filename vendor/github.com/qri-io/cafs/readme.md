# cafs

[![Qri](https://img.shields.io/badge/made%20by-qri-magenta.svg?style=flat-square)](https://qri.io)
[![GoDoc](https://godoc.org/github.com/qri-io/cafs?status.svg)](http://godoc.org/github.com/qri-io/cafs)
[![License](https://img.shields.io/github/license/qri-io/cafs.svg?style=flat-square)](./LICENSE)
<!-- [![Codecov](https://img.shields.io/codecov/c/github/qri-io/cafs.svg?style=flat-square)](https://codecov.io/gh/qri-io/cafs)
[![CI](https://img.shields.io/circleci/project/github/qri-io/cafs.svg?style=flat-square)](https://circleci.com/gh/qri-io/cafs) -->

`cafs` stands for "content-addressed-file-system", which is a generalized interface for working with filestores that determine names content based on the content itself. Examples of a content-addressed file systems include git, IPFS, the DAT project, and like, blockchains or whatever. The long-term goal of cafs is to decouple & interoperate _common filestore operations_ between different content-addressed filestores. This package doesn't aim to implement everything a given filestore can do, but instead focus on basic file & directory i/o. cafs is very early days, starting with a proof of concept based on IPFS and an in-memory implementation. Over time we'll work to add additional stores, which will undoubtably affect the overall interface definition.

## Getting Involved

We would love involvement from more people! If you notice any errors or would
like to submit changes, please see our
[Contributing Guidelines](./.github/CONTRIBUTING.md).