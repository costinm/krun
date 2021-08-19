ARG BASE=gcr.io/istio-testing/proxyv2:latest

FROM golang:latest AS build

#FROM golang:alpine AS build-base
# dlv doesn't seem to work yet ?

WORKDIR /ws
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOPROXY=https://proxy.golang.org

COPY go.* ./
# Helps speed up local builds
#RUN go mod download

COPY *.go ./
COPY cmd ./cmd/
COPY pkg ./pkg/

RUN go build -a -gcflags='all=-N -l' -ldflags '-extldflags "-static"' -o /ws/krun ./

RUN go build -a -gcflags='all=-N -l' -ldflags '-extldflags "-static"' -o /ws/hbgate ./cmd/gate

FROM ${BASE} AS istio

FROM ubuntu:bionic

# Similar with the 'ko' runtime layout
COPY --from=build /ws/krun /ko-app/krun
COPY --from=build /ws/hbgate /usr/local/bin/hbgate
COPY --from=istio /usr/local/bin/pilot-agent /usr/local/bin/

WORKDIR /

ENTRYPOINT ["/ko-app/krun"]
