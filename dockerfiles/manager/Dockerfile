# syntax = docker/dockerfile:1.4

# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the go source
COPY . .

# Build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    make build/manager

FROM gcr.io/distroless/static-debian11
WORKDIR /
COPY --from=builder /workspace/bin/manager .

ENTRYPOINT ["/manager"]
