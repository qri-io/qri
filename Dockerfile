FROM golang:1.9-alpine
ADD . /go/src/github.com/qri-io/qri
RUN go install github.com/qri-io/qri

ENV PORT 8080
# Set binary as entrypoint
CMD ["./qri", "server"]