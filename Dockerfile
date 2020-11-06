# build image to construct the binary
FROM golang:1.14.6-alpine AS builder
LABEL maintainer="sparkle_pony_2000@qri.io"

RUN apk update && apk upgrade && \
apk add --no-cache bash git openssh

# RUN apk add --no-cache autoconf automake libtool gettext gettext-dev make g++ texinfo curl
# WORKDIR /root
# RUN wget https://github.com/emcrisostomo/fswatch/releases/download/1.14.0/fswatch-1.14.0.tar.gz
# RUN tar -xvzf fswatch-1.14.0.tar.gz
# WORKDIR /root/fswatch-1.14.0
# RUN ./configure
# RUN make && make install

# build environment variables:
#   * enable go modules
#   * use goproxy
#   * disable cgo for our builds
#   * ensure target os is linux
ENV GO111MODULE=on \
  GOPROXY=https://proxy.golang.org \
  CGO_ENABLED=0 \
  GOOS=linux

# need to update to latest ca-certificates, otherwise TLS won't work properly.
# Informative:
# https://hackernoon.com/alpine-docker-image-with-secured-communication-ssl-tls-go-restful-api-128eb6b54f1f
RUN apk update \
    && apk upgrade \
    && apk add --no-cache \
    ca-certificates \
    && update-ca-certificates 2>/dev/null || true

# add local files to cloud backend
ADD . /qri
WORKDIR /qri

# install to produce a binary called "main" in the pwd
#   -a flag rebuild all the packages we’re using,
#      which means all the imports will be rebuilt with cgo disabled.
#   -installsuffix cgo keeps output separate in build caches
#   -o sets the output name to main
#   . says "build this package"
RUN go build -a -installsuffix cgo -o main .

# *** production image ***
# use alpine as base for smaller images
FROM alpine:latest as production
LABEL maintainer="sparkle_pony_2000@qri.io" 

# create directories for IPFS & QRI, setting proper owners
RUN mkdir -p $QRI_PATH /app

WORKDIR /app

# Copy our static executable, qri and IPFS directories, which are empty
COPY --from=builder /qri/main /bin/qri

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