
default: build

require-goversion:
	$(eval minver := go1.12)
# Get the version of the current go binary
	$(eval havever := $(shell go version | awk '{print $$3}'))
# Magic happens. Sort using "." as the tab, keyed by groups of numbers,
# take the smallest.
	$(eval match := $(shell echo "$(minver)\n$(havever)" | sort -t '.' -k 1,1 -k 2,2 -g -r | head -n 1))
# If the minimum version either matches exactly what we have, or does not match
# the result of the magic sort above, we're okay. Otherwise, our binary's
# version isn't good enough: error.
	@if [ "$(havever)" != "$(minver)" ]; then \
		if [ "$(match)" == "$(minver)" ]; then \
			echo "Error: invalid go version $(havever), need $(minver)"; exit 1; \
		fi; \
	fi;

build: require-goversion
	go install

build-latest:
	git checkout master && git pull
	build

update-qri-deps: require-gopath
	cd $$GOPATH/src/github.com/qri-io/qri && git checkout master && git pull && gx install
	cd $$GOPATH/src/github.com/qri-io/qfs && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/registry && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/dataset && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/varName && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/deepdiff && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/jsonschema && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/starlib && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/dag && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/ioes && git checkout master && git pull
	cd $$GOPATH/src/github.com/qri-io/qri

workdir:
	mkdir -p workdir

lint:
	golint ./...

test:
	go test ./... -v --coverprofile=coverage.txt --covermode=atomic

test-all-coverage:
	./.circleci/cover.test.sh

cli-docs:
	go run docs/docs.go --dir temp --filename cli_commands.md

update-changelog:
	conventional-changelog -p angular -i CHANGELOG.md -s

build-cross-platform:
	@echo "building qri_windows_amd64"
	mkdir qri_windows_amd64
	env GOOS=windows GOARCH=amd64 go build -o qri_windows_amd64/qri .
	zip -r qri_windows_amd64.zip qri_windows_amd64 && rm -r qri_windows_amd64
	@echo "building qri_windows_386"
	mkdir qri_windows_386
	env GOOS=windows GOARCH=386 go build -o qri_windows_386/qri .
	zip -r qri_windows_386.zip qri_windows_386 && rm -r qri_windows_386
	@echo "building qri_linux_arm"
	mkdir qri_linux_arm
	env GOOS=linux GOARCH=arm go build -o qri_linux_arm/qri .
	zip -r qri_linux_arm.zip qri_linux_arm && rm -r qri_linux_arm
	@echo "building qri_linux_amd64"
	mkdir qri_linux_amd64
	env GOOS=linux GOARCH=amd64 go build -o qri_linux_amd64/qri .
	zip -r qri_linux_amd64.zip qri_linux_amd64 && rm -r qri_linux_amd64
	@echo "building qri_linux_386"
	mkdir qri_linux_386
	env GOOS=linux GOARCH=386 go build -o qri_linux_386/qri .
	zip -r qri_linux_386.zip qri_linux_386 && rm -r qri_linux_386
	@echo "building qri_darwin_386"
	mkdir qri_darwin_386
	env GOOS=darwin GOARCH=386 go build -o qri_darwin_386/qri .
	zip -r qri_darwin_386.zip qri_darwin_386 && rm -r qri_darwin_386	
	@echo "building qri_darwin_amd64"
	mkdir qri_darwin_amd64
	env GOOS=darwin GOARCH=amd64 go build -o qri_darwin_amd64/qri .
	zip -r qri_darwin_amd64.zip qri_darwin_amd64 && rm -r qri_darwin_amd64