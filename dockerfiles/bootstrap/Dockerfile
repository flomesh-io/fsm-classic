# syntax = docker/dockerfile:1.4

# Build the bootstrap binary
FROM golang:1.18-alpine3.15 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/go/pkg \
    go mod download

# Copy the go source
COPY . .

# Build
RUN apk add make bash git
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    make build/bootstrap

FROM alpine:3.15
WORKDIR /
COPY --from=builder /workspace/bin/bootstrap .

ENTRYPOINT ["/bootstrap"]
