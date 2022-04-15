# Build the bootstrap binary
FROM golang:1.17-alpine3.14 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# Build
RUN apk add --update make bash git && make build/bootstrap

FROM alpine:3.14
WORKDIR /
COPY --from=builder /workspace/bin/bootstrap .

ENTRYPOINT ["/bootstrap"]
