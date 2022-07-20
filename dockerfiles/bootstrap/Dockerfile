# syntax = docker/dockerfile:1.4

# Build the bootstrap binary
FROM golang:1.18 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the go source
COPY . .

# Build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH make build/bootstrap

FROM gcr.io/distroless/static-debian11
WORKDIR /
COPY --from=builder /workspace/bin/bootstrap .

ENTRYPOINT ["/bootstrap"]
