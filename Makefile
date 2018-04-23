GOFILES = $(shell find . -name '*.go' -not -path './vendor/*')
GOPACKAGES = github.com/briandowns/spinner github.com/datatogether/api/apiutil github.com/fatih/color github.com/ipfs/go-datastore github.com/olekukonko/tablewriter github.com/qri-io/analytics github.com/qri-io/bleve github.com/qri-io/dataset github.com/qri-io/doggos github.com/qri-io/dsdiff github.com/qri-io/varName github.com/qri-io/registry/regclient github.com/sirupsen/logrus github.com/spf13/cobra github.com/spf13/cobra/doc github.com/ugorji/go/codec github.com/theckman/go-flock

default: build

require-gopath:
ifndef GOPATH
	$(error $GOPATH must be set. plz check: https://github.com/golang/go/wiki/SettingGOPATH)
endif

build: require-gopath
	@echo ""
	@echo "1/5 install non-gx deps:"
	@echo ""
	go get -v -u $(GOPACKAGES)
	@echo ""
	@echo "2/5 install gx:"
	@echo ""
	go get -v -u github.com/whyrusleeping/gx github.com/whyrusleeping/gx-go
	@echo ""
	@echo "3/5 install gx deps:"
	@echo ""
	$$GOPATH/bin/gx install
	@echo ""
	@echo ""
	@echo "4/5 install gx dep-packages:"
	@echo ""
	go get github.com/qri-io/cafs
	@echo ""
	@echo "5/5 build & install qri:"
	@echo ""
	go install
	@echo "done!"

update-qri-deps: require-gopath
	cd $$GOPATH/src/github.com/qri-io/qri && git checkout master && git pull && gx install
	cd $$GOPATH/src/github.com/qri-io/cafs && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/dataset && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/varName && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/dsdiff && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/jsonschema && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/qri

install-deps:
	go get -v -u $(GOPACKAGES)

install-gx:
	go get -v -u github.com/whyrusleeping/gx github.com/whyrusleeping/gx-go

install-gx-deps:
	gx install

workdir:
	mkdir -p workdir

lint:
	golint ./...

test:
	go test ./...

test-all-coverage:
	./.circleci/cover.test.sh