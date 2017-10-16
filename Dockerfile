FROM golang:1.9-alpine

ADD . /go/src/github.com/qri-io/qri
RUN go install github.com/qri-io/qri

# default port is 8080
ENV PORT 8080
# default log level=
ENV IPFS_LOGGING ""
# create qri path
ENV QRI_PATH /data/qri
# Create the fs-repo directory and switch to a non-privileged user.
ENV IPFS_PATH /data/ipfs

# Ports for Swarm TCP, Swarm uTP, API, Gateway, Swarm Websockets
EXPOSE 4001
EXPOSE 4002/udp
EXPOSE 5001
EXPOSE 8080
EXPOSE 8081

RUN mkdir -p $IPFS_PATH \
  && adduser -D -h $IPFS_PATH -u 1000 -g 100 ipfs \
  && chown 1000:100 $IPFS_PATH

# Expose the fs-repo as a volume.
# start_ipfs initializes an fs-repo if none is mounted.
# Important this happens after the USER directive so permission are correct.
VOLUME $IPFS_PATH

VOLUME $QRI_PATH

# Set binary as entrypoint, initalizing ipfs repo if none is mounted
CMD ["./qri", "server", "--init-ipfs"]