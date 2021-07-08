
default: build

QRI_VERSION?="0.10.1-dev"

BUILD_FLAGS?=CGO_ENABLED=0
PKG=$(shell go list ./version)
GOLANG_VERSION=$(shell go version | awk '{print $$3}')

require-goversion:
# Parse version from go.mod, remove space so it matches `go version`
	$(eval minver := $(shell grep "^go" go.mod | tr -s 'go ' 'go'))
# Magic happens. Sort using "." as the tab, keyed by groups of numbers,
# take the smallest.
	$(eval match := $(shell echo "$(minver)\n$(GOLANG_VERSION)" | sort -t '.' -k 1,1 -k 2,2 -g -r | head -n 1))
# If the minimum version either matches exactly what we have, or does not match
# the result of the magic sort above, we're okay. Otherwise, our binary's
# version isn't good enough: error.
	@if [ "$(GOLANG_VERSION)" != "$(minver)" ]; then \
		if [ "$(match)" == "$(minver)" ]; then \
			echo "Error: invalid go version $(GOLAND_VERSION), need $(minver)"; exit 1; \
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

serve-api-docs:
	cd docs && go run . --http :2502

update-changelog:
	conventional-changelog -p angular -i CHANGELOG.md -s
