module github.com/qri-io/qri

go 1.12

// See https://github.com/dgraph-io/badger/issues/904

//replace github.com/dgraph-io/badger v2.0.0-rc.2+incompatible => github.com/dgraph-io/badger/v2 v2.0.0-rc.2

//replace github.com/dgraph-io/badger/v2 v2.0.0-rc2 => github.com/dgraph-io/badger v1.6.0-rc1

require (
	cloud.google.com/go v0.43.0 // indirect
	github.com/beme/abide v0.0.0-20181227202223-4c487ef9d895
	github.com/btcsuite/goleveldb v1.0.0 // indirect
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/dgraph-io/badger v2.0.0-rc.2+incompatible // indirect
	github.com/dgraph-io/badger/v2 v2.0.0-rc2 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-kit/kit v0.9.0 // indirect
	github.com/gofrs/flock v0.7.1 // indirect
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/google/flatbuffers v1.11.0
	github.com/google/go-cmp v0.3.0
	github.com/google/pprof v0.0.0-20190723021845-34ac40c74b70 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.5 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/ipfs/go-cid v0.0.2
	github.com/ipfs/go-ipfs v0.4.22-rc1
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/interface-go-ipfs-core v0.0.8
	github.com/jessevdk/go-flags v1.4.0 // indirect
	github.com/kisielk/errcheck v1.2.0 // indirect
	github.com/kkdai/bstream v0.0.0-20181106074824-b3251f7901ec // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/libp2p/go-libp2p v0.0.28
	github.com/libp2p/go-libp2p-circuit v0.0.8
	github.com/libp2p/go-libp2p-connmgr v0.0.6
	github.com/libp2p/go-libp2p-crypto v0.0.2
	github.com/libp2p/go-libp2p-host v0.0.3
	github.com/libp2p/go-libp2p-net v0.0.2
	github.com/libp2p/go-libp2p-peer v0.1.1
	github.com/libp2p/go-libp2p-peerstore v0.0.6
	github.com/libp2p/go-libp2p-protocol v0.0.1
	github.com/libp2p/go-libp2p-swarm v0.0.6
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/mr-tron/base58 v1.1.2
	github.com/multiformats/go-multiaddr v0.0.4
	github.com/multiformats/go-multicodec v0.1.6
	github.com/multiformats/go-multihash v0.0.5
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/qri-io/apiutil v0.1.0
	github.com/qri-io/bleve v0.5.1-0.20190530204435-e47ddda1936d
	github.com/qri-io/dag v0.1.0
	github.com/qri-io/dataset v0.1.2
	github.com/qri-io/deepdiff v0.1.0
	github.com/qri-io/doggos v0.1.0
	github.com/qri-io/ioes v0.1.0
	github.com/qri-io/iso8601 v0.1.0
	github.com/qri-io/jsonschema v0.1.1
	github.com/qri-io/qfs v0.1.1-0.20190807153257-0225cf9fd84e
	github.com/qri-io/registry v0.1.0
	github.com/qri-io/starlib v0.4.1
	github.com/qri-io/varName v0.1.0
	github.com/rogpeppe/fastuuid v1.2.0 // indirect
	github.com/russross/blackfriday v2.0.0+incompatible // indirect
	github.com/sergi/go-diff v1.0.0
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.4.0 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/theckman/go-flock v0.7.1
	github.com/ugorji/go v1.1.7 // indirect
	go.etcd.io/bbolt v1.3.3 // indirect
	go.starlark.net v0.0.0-20190528202925-30ae18b8564f
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/mobile v0.0.0-20190806162312-597adff16ade // indirect
	golang.org/x/net v0.0.0-20190724013045-ca1201d0de80 // indirect
	golang.org/x/sys v0.0.0-20190804053845-51ab0e2deafa
	golang.org/x/tools v0.0.0-20190806215303-88ddfcebc769 // indirect
	google.golang.org/genproto v0.0.0-20190801165951-fa694d86fc64 // indirect
	google.golang.org/grpc v1.22.1 // indirect
	gopkg.in/yaml.v2 v2.2.2
	honnef.co/go/tools v0.0.1-2019.2.2 // indirect
)
