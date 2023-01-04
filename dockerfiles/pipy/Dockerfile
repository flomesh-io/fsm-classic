# syntax = docker/dockerfile:1.4
ARG DISTROLESS_TAG

# Build the pipy binary
FROM --platform=$BUILDPLATFORM node:16-bullseye AS builder

ARG PIPY_VERSION

RUN apt-get update && \
    apt-get install --yes --no-install-recommends \
      cmake clang pkgconf nasm

WORKDIR /workspace
RUN git clone -b "$PIPY_VERSION" https://github.com/flomesh-io/pipy.git && \
    cd /workspace/pipy && ./build.sh -s

# Build the final image
FROM gcr.io/distroless/base-nossl-debian11:$DISTROLESS_TAG
WORKDIR /
COPY --from=builder /workspace/pipy/bin/pipy /usr/local/bin/pipy
USER 65532:65532

ENTRYPOINT ["/usr/local/bin/pipy"]
