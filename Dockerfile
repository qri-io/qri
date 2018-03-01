FROM golang:1.9
LABEL maintainer="sparkle_pony_2000@qri.io"

ADD . /go/src/github.com/qri-io/qri
RUN cd /go/src/github.com/qri-io/qri

# RUN make install-deps
RUN go get -v github.com/briandowns/spinner github.com/datatogether/api/apiutil github.com/fatih/color github.com/ipfs/go-datastore github.com/olekukonko/tablewriter github.com/qri-io/analytics github.com/qri-io/bleve github.com/qri-io/dataset github.com/qri-io/doggos github.com/sirupsen/logrus github.com/spf13/cobra github.com/spf13/viper github.com/qri-io/varName github.com/qri-io/dsdiff github.com/datatogether/cdxj github.com/qri-io/cafs

RUN go get -u github.com/whyrusleeping/gx github.com/whyrusleeping/gx-go
RUN cd /go/src/github.com/qri-io/qri && pwd && gx install

RUN go install github.com/qri-io/qri
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