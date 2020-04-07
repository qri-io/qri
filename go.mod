module github.com/qri-io/qri

go 1.12

replace (
	github.com/go-critic/go-critic v0.0.0-20181204210945-c3db6069acc5 => github.com/go-critic/go-critic v0.0.0-20190422201921-c3db6069acc5
	github.com/go-critic/go-critic v0.0.0-20181204210945-ee9bf5809ead => github.com/go-critic/go-critic v0.0.0-20190210220443-ee9bf5809ead
	github.com/golangci/errcheck v0.0.0-20181003203344-ef45e06d44b6 => github.com/golangci/errcheck v0.0.0-20181223084120-ef45e06d44b6
	github.com/golangci/go-tools v0.0.0-20180109140146-af6baa5dc196 => github.com/golangci/go-tools v0.0.0-20190318060251-af6baa5dc196
	github.com/golangci/gofmt v0.0.0-20181105071733-0b8337e80d98 => github.com/golangci/gofmt v0.0.0-20181222123516-0b8337e80d98
	github.com/golangci/gosec v0.0.0-20180901114220-66fb7fc33547 => github.com/golangci/gosec v0.0.0-20190211064107-66fb7fc33547
	github.com/golangci/lint-1 v0.0.0-20180610141402-ee948d087217 => github.com/golangci/lint-1 v0.0.0-20190420132249-ee948d087217
	mvdan.cc/unparam v0.0.0-20190124213536-fbb59629db34 => mvdan.cc/unparam v0.0.0-20190209190245-fbb59629db34
)

require (
	github.com/beme/abide v0.0.0-20181227202223-4c487ef9d895
	github.com/cube2222/octosql v0.2.1-0.20200319150444-e5a71fa20dbe
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.9.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/gofrs/flock v0.7.1 // indirect
	github.com/google/flatbuffers v1.11.0
	github.com/google/go-cmp v0.3.1
	github.com/ipfs/go-cid v0.0.3
	github.com/ipfs/go-datastore v0.1.1
	github.com/ipfs/go-ds-badger v0.0.7 // indirect
	github.com/ipfs/go-ipfs v0.4.22-0.20191023033800-4a102207a36c
	github.com/ipfs/go-ipfs-config v0.0.11
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/interface-go-ipfs-core v0.2.3
	github.com/jinzhu/copier v0.0.0-20180308034124-7e38e58719c3
	github.com/libp2p/go-libp2p v0.4.0
	github.com/libp2p/go-libp2p-circuit v0.1.3
	github.com/libp2p/go-libp2p-connmgr v0.1.1
	github.com/libp2p/go-libp2p-core v0.2.3
	github.com/libp2p/go-libp2p-peerstore v0.1.3
	github.com/libp2p/go-libp2p-swarm v0.2.2
	github.com/microcosm-cc/bluemonday v1.0.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mr-tron/base58 v1.1.2
	github.com/multiformats/go-multiaddr v0.1.1
	github.com/multiformats/go-multicodec v0.1.6
	github.com/multiformats/go-multihash v0.0.8
	github.com/pkg/errors v0.8.1
	github.com/qri-io/apiutil v0.1.0
	github.com/qri-io/dag v0.2.1-0.20191025201336-254aa177fbd7
	github.com/qri-io/dataset v0.1.5-0.20200324184139-108a69072ede
	github.com/qri-io/deepdiff v0.1.1-0.20200305020550-8173efebcaa1
	github.com/qri-io/doggos v0.1.0
	github.com/qri-io/ioes v0.1.1
	github.com/qri-io/iso8601 v0.1.0
	github.com/qri-io/jsonschema v0.1.1
	github.com/qri-io/qfs v0.1.1-0.20200207165332-a6a7b98f0fa0
	github.com/qri-io/starlib v0.4.2-0.20200406084526-0e4566b610a1
	github.com/qri-io/varName v0.1.0
	github.com/russross/blackfriday/v2 v2.0.2-0.20190629151518-3e56bb68c887
	github.com/sergi/go-diff v1.0.0
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/cobra v0.0.5
	github.com/theckman/go-flock v0.7.1
	github.com/ugorji/go/codec v1.1.7
	go.starlark.net v0.0.0-20200330013621-be5394c419b6
	golang.org/x/crypto v0.0.0-20190926180335-cea2066c6411
	golang.org/x/sys v0.0.0-20200323222414-85ca7c5b95cd
	gonum.org/v1/gonum v0.6.0
	gopkg.in/yaml.v2 v2.2.8
	nhooyr.io/websocket v1.7.4
)
