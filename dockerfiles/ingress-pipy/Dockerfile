# syntax = docker/dockerfile:1.4

# Build the ingress-pipy binary
FROM --platform=$BUILDPLATFORM golang:1.18-alpine3.15 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the go source
COPY . .

# Build
RUN apk add make bash git
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH make build/ingress-pipy

FROM flomesh/pipy:0.50.0-25
WORKDIR /
COPY --from=builder /workspace/bin/ingress-pipy .

ENTRYPOINT ["/ingress-pipy"]
