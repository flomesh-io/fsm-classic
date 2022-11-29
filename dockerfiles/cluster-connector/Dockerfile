# syntax = docker/dockerfile:1.4
ARG TARGETOS
ARG TARGETARCH

# Build the cluster-connector binary
FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi8/go-toolset:1.17 as builder

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
RUN microdnf update && microdnf install make git
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH make build/cluster-connector

FROM registry.access.redhat.com/ubi8-minimal
WORKDIR /
COPY --from=builder /workspace/bin/cluster-connector .
#COPY --from=builder /workspace/config/cluster-connector/connector_config.yaml .

#RUN apk --no-cache add curl tzdata

ENTRYPOINT ["/cluster-connector"]
