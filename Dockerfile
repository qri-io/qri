# build image to construct the binary
FROM golang:1.14.6-alpine AS builder
LABEL maintainer="sparkle_pony_2000@qri.io"

RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
       bash git make openssh

# build environment variables:
#   * enable go modules
#   * use goproxy
#   * disable cgo for our builds
#   * ensure target os is linux
ENV GO111MODULE=on \
    GOPROXY=https://proxy.golang.org \
    CGO_ENABLED=0 \
    GOOS=linux

# add sorce code to a "/qri" directory on the build image
ADD . /qri
# use that directory for working
WORKDIR /qri

# build using make command
RUN make build

# *** production image ***
# use alpine as base for smaller images
FROM alpine:latest as production
LABEL maintainer="sparkle_pony_2000@qri.io" 

# create directories for IPFS & QRI, setting proper owners
RUN mkdir -p $QRI_PATH /app

WORKDIR /app

# Copy our static executable from the builder image
COPY --from=builder /qri/qri /bin/qri

# need to update to latest ca-certificates, otherwise TLS won't work properly.
# Informative:
# https://hackernoon.com/alpine-docker-image-with-secured-communication-ssl-tls-go-restful-api-128eb6b54f1f
RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
       ca-certificates \
    && update-ca-certificates 2>/dev/null || true

# Set binary as entrypoint
# the setup flag initalizes ipfs & qri repos if none is mounted
# the migrate flag automatically executes any necessary migrations
# the no-prompt flag disables asking for any user input, reverting to defaults
CMD ["qri", "connect", "--setup", "--migrate", "--no-prompt"]