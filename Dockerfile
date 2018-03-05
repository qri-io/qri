FROM golang:1.10
LABEL maintainer="sparkle_pony_2000@qri.io"

ADD . /go/src/github.com/qri-io/qri

# run build
RUN cd /go/src/github.com/qri-io/qri && make build

# set default port to 8080, default log level, QRI_PATH env, IPFS_PATH env
ENV PORT=8080 IPFS_LOGGING="" QRI_PATH=/data/qri IPFS_PATH=/data/ipfs

# Ports for Swarm TCP, Swarm uTP, API, Gateway, Swarm Websockets
EXPOSE 4001 4002/udp 5001 8080 8081

# create directories for IPFS & QRI, setting proper owners
RUN mkdir -p $IPFS_PATH && mkdir -p $QRI_PATH \
  && adduser -D -h $IPFS_PATH -u 1000 -g 100 ipfs \
  && chown 1000:100 $IPFS_PATH \
  && chown 1000:100 $QRI_PATH

# Expose the fs-repo & qri-repos as volumes.
# Important this happens after the USER directive so permission are correct.
# VOLUME $IPFS_PATH
# VOLUME $QRI_PATH

# Set binary as entrypoint, initalizing ipfs & qri repos if none is mounted
CMD ["qri", "connect", "--setup"]