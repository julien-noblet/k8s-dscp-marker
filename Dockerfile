# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/k8s-dscp-marker .

FROM alpine:3.20
RUN apk add --no-cache iptables
COPY --from=build /out/k8s-dscp-marker /usr/local/bin/k8s-dscp-marker
ENTRYPOINT ["/usr/local/bin/k8s-dscp-marker"]
