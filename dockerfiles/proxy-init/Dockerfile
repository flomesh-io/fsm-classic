# syntax = docker/dockerfile:1.4
ARG TARGETOS
ARG TARGETARCH

# Build the proxy-init binary
FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi8-minimal as builder

RUN microdnf update && microdnf install tar gzip make git
RUN curl -L -O https://go.dev/dl/go1.17.13.$TARGETOS-$TARGETARCH.tar.gz
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.13.$TARGETOS-$TARGETARCH.tar.gz
RUN export PATH=$PATH:/usr/local/go/bin

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
#RUN apk add make bash git
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH make build/proxy-init

FROM registry.access.redhat.com/ubi8-minimal
WORKDIR /
COPY --from=builder /workspace/bin/proxy-init .

ENTRYPOINT ["/proxy-init"]