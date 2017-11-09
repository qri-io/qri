# go-log

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![standard-readme compliant](https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![GoDoc](https://godoc.org/github.com/ipfs/go-log?status.svg)](https://godoc.org/github.com/ipfs/go-log)
[![Build Status](https://travis-ci.org/ipfs/go-log.svg?branch=master)](https://travis-ci.org/ipfs/go-log)

<!---[![Coverage Status](https://coveralls.io/repos/github/ipfs/go-log/badge.svg?branch=master)](https://coveralls.io/github/ipfs/go-log?branch=master)--->


> The logging library used by go-ipfs

It currently uses a modified version of [go-logging](https://github.com/whyrusleeping/go-logging) to implement the standard printf-style log output.

## Install

```sh
go get github.com/ipfs/go-log
```

## Usage

Once the pacakge is imported under the name `logging`, an instance of `EventLogger` can be created like so:

```go
var log = logging.Logger("subsystem name")
```

It can then be used to emit log messages, either plain printf-style messages at six standard levels or structured messages using `Event` and `EventBegin` methods.

## Contribute

Feel free to join in. All welcome. Open an [issue](https://github.com/ipfs/go-log/issues)!

This repository falls under the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

### Want to hack on IPFS?

[![](https://cdn.rawgit.com/jbenet/contribute-ipfs-gif/master/img/contribute.gif)](https://github.com/ipfs/community/blob/master/contributing.md)

## License

MIT
