# syntax = docker/dockerfile:1.4
ARG TARGETOS
ARG TARGETARCH
ARG DISTROLESS_TAG

# Build the pipy binary
FROM --platform=$BUILDPLATFORM debian:11-slim AS builder-pipy

WORKDIR /workspace

RUN apt-get update && \
    apt-get install --yes git curl openssh-client cmake clang autoconf automake libtool pkgconf nasm file && \
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.3/install.sh | bash && \
    export NVM_DIR="$HOME/.nvm" && \
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" && \
    [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion" && \
    nvm install --lts=Gallium && \
    git clone -b 0.70.0-46 https://github.com/flomesh-io/pipy.git && \
    cd /workspace/pipy && ./build.sh -s

# Build the ingress-pipy binary
FROM --platform=$BUILDPLATFORM golang:1.19-alpine3.15 as builder-go

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/go/pkg go mod download
# Copy the go source
COPY . .

# Build
RUN apk add make bash git
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH make build/ingress-pipy

# Build the final image
FROM gcr.io/distroless/base-debian11:$DISTROLESS_TAG
WORKDIR /
COPY --from=builder-go /workspace/bin/ingress-pipy .
COPY --from=builder-pipy /workspace/pipy/bin/pipy /usr/local/bin/pipy

USER 65532:65532

ENTRYPOINT ["/ingress-pipy"]
