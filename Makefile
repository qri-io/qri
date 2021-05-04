
default: build

QRI_VERSION?="0.10.0"

BUILD_FLAGS?=CGO_ENABLED=0
PKG=$(shell go list ./version)
GOLANG_VERSION=$(shell go version | awk '{print $$3}')

require-goversion:
	$(eval minver := go1.13)
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

require-govvv:
ifeq (,$(shell which govvv))
	@echo "installing govvv"
	$(shell go install github.com/ahmetb/govvv)
endif

build: require-goversion require-govvv
	$(BUILD_FLAGS) go build -ldflags="-X ${PKG}.GolangVersion=${GOLANG_VERSION} $(shell govvv -flags -pkg $(PKG) -version $(QRI_VERSION))" .

install: require-goversion require-govvv
	$(BUILD_FLAGS) go install -ldflags="-X ${PKG}.GolangVersion=${GOLANG_VERSION} $(shell govvv -flags -pkg $(PKG) -version $(QRI_VERSION))" .
.PHONY: install

dscache_fbs:
	cd dscache && flatc --go def.fbs

workdir:
	mkdir -p workdir

lint:
	golint ./...

test:
	go test ./... -v --coverprofile=coverage.txt --covermode=atomic

test-all-coverage:
	./.circleci/cover.test.sh

cli-docs:
	cd docs && go run . --dir ../temp --filename cli_commands.md

api-spec:
	cd docs && go run . --dir ../temp --apiOnly

update-changelog:
	conventional-changelog -p angular -i CHANGELOG.md -s
