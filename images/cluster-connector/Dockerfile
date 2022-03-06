# Build the manager binary
FROM golang:1.17-alpine3.14 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# Build
#RUN go env GOOS GOARCH && CGO_ENABLED=0 GO111MODULE=on GOPROXY=https://goproxy.io go build -a -o manager main.go
RUN apk add --update make bash git && make build/cluster-connector

FROM alpine:3.14
WORKDIR /
COPY --from=builder /workspace/bin/cluster-connector .
#COPY --from=builder /workspace/config/cluster-connector/connector_config.yaml .

RUN apk --no-cache add curl tzdata

ENTRYPOINT ["/cluster-connector"]
