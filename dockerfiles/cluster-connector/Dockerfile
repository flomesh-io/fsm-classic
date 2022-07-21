# syntax = docker/dockerfile:1.4
ARG TARGETOS
ARG TARGETARCH
ARG DISTROLESS_TAG

# Build the cluster-connector binary
FROM --platform=$BUILDPLATFORM golang:1.18 as builder

WORKDIR /workspace
# Copy the go source
COPY . .

# Build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH make build/cluster-connector

FROM gcr.io/distroless/static-debian11:$DISTROLESS_TAG
WORKDIR /
COPY --from=builder /workspace/bin/cluster-connector .

ENTRYPOINT ["/cluster-connector"]
