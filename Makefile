GOFILES = $(shell find . -name '*.go' -not -path './vendor/*')
GOPACKAGES = $(shell go list ./...  | grep -v /vendor/ | grep qri/core)

default: build

workdir:
	mkdir -p workdir

build: workdir/qri

build-native: $(GOFILES)
	go build -o workdir/native-qri .

workdir/qri: $(GOFILES)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o workdir/qri .

test: test-all

test-all:
	./.circleci/cover.test.sh