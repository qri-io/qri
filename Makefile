GOFILES = $(shell find . -name '*.go' -not -path './vendor/*')
GOPACKAGES = $(shell go list ./...  | grep -v /vendor/ | grep qri/core)

default: build

build:
	@# install non-gx deps
	go get -v github.com/briandowns/spinner github.com/datatogether/api/apiutil github.com/fatih/color github.com/ipfs/go-datastore github.com/olekukonko/tablewriter github.com/qri-io/analytics github.com/qri-io/bleve github.com/qri-io/dataset github.com/qri-io/doggos github.com/sirupsen/logrus github.com/spf13/cobra github.com/spf13/viper github.com/qri-io/varName github.com/qri-io/datasetDiffer github.com/datatogether/cdxj
	@# install gx
	go get -v -u github.com/whyrusleeping/gx github.com/whyrusleeping/gx-go
	@# install gx deps. This will take time.
	gx install
	@# install deps with gx depenencies
	go get github.com/qri-io/cafs
	@# build the thing
	go build

install-deps:
	go get -v github.com/briandowns/spinner github.com/datatogether/api/apiutil github.com/fatih/color github.com/ipfs/go-datastore github.com/olekukonko/tablewriter github.com/qri-io/analytics github.com/qri-io/bleve github.com/qri-io/dataset github.com/qri-io/doggos github.com/sirupsen/logrus github.com/spf13/cobra github.com/spf13/viper github.com/qri-io/varName github.com/qri-io/datasetDiffer github.com/datatogether/cdxj

install-gx:
	go get -v -u github.com/whyrusleeping/gx github.com/whyrusleeping/gx-go

install-gx-deps:
	gx install

workdir:
	mkdir -p workdir

# build: workdir/qri

# build-native: $(GOFILES)
# 	go build -o workdir/native-qri .

# workdir/qri: $(GOFILES)
	# GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o workdir/qri .

lint:
	golint ./...

test:
	go test ./...

test-all-coverage:
	./.circleci/cover.test.sh