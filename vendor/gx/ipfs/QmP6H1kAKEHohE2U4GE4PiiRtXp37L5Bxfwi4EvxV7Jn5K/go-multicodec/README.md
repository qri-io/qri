# go-multicodec

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-multiformats-blue.svg?style=flat-square)](http://github.com/multiformats/multiformats)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)

> multicodec: self-describing serialization

This is the [multicodec](https://github.com/multiformats/multicodec) implementation in Go.

### Supported codecs

- `/protobuf`
- `/cbor`
- `/json`

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Maintainers](#maintainers)
- [Contribute](#contribute)
- [License](#license)

## Install

```sh
go get github.com/multiformats/go-multicodec
```

## Usage

Look at the Godocs:

- https://godoc.org/github.com/jbenet/multicodec

```go
import (
  "os"
  "io"

  cbor "github.com/jbenet/go-multicodec/cbor"
  json "github.com/jbenet/go-multicodec/json"
)

func main() {
  dec := cbor.Multicodec().NewDecoder(os.Stdin)
  enc := json.Multicodec().NewEncoder(os.Stdout)

  for {
    var item interface{}

    if err := dec.Decode(&item); err == io.EOF {
      break
    } else if err != nil {
      panic(err)
    }

    if err := enc.Encode(&item); err != nil {
      panic(err)
    }
  }
}
```

## Maintainers

Captain: [@jbenet](https://github.com/jbenet).

## Contribute

Contributions welcome. Please check out [the issues](https://github.com/multiformats/go-multicodec/issues).

Check out our [contributing document](https://github.com/multiformats/multiformats/blob/master/contributing.md) for more information on how we work, and about contributing in general. Please be aware that all interactions related to multiformats are subject to the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

[MIT](LICENSE) © Juan Batiz-Benet
