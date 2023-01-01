# syntax = docker/dockerfile:1.4
ARG DISTROLESS_TAG

# Build the pipy binary
FROM --platform=$BUILDPLATFORM debian:11-slim AS builder-pipy

WORKDIR /workspace
ARG PIPY_VERSION

RUN apt-get update && \
    apt-get install --yes git curl openssh-client cmake clang autoconf automake libtool pkgconf nasm file && \
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.3/install.sh | bash && \
    export NVM_DIR="$HOME/.nvm" && \
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" && \
    [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion" && \
    nvm install --lts=Gallium

RUN git clone -b "$PIPY_VERSION" https://github.com/flomesh-io/pipy.git && \
    cd /workspace/pipy && ./build.sh -s


# Build the final image
FROM gcr.io/distroless/base-debian11:$DISTROLESS_TAG
WORKDIR /
COPY --from=builder-pipy /workspace/pipy/bin/pipy /usr/local/bin/pipy
USER 65532:65532

ENTRYPOINT ["/usr/local/bin/pipy"]
