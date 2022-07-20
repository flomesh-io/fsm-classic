# syntax = docker/dockerfile:1.4

# Build the cluster-connector binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the go source
COPY . .

# Build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    make build/cluster-connector

FROM gcr.io/distroless/static-debian11
WORKDIR /
COPY --from=builder /workspace/bin/cluster-connector .
#COPY --from=builder /workspace/config/cluster-connector/connector_config.yaml .

RUN apk --no-cache add curl tzdata

ENTRYPOINT ["/cluster-connector"]
