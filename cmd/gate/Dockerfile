ARG BASE=gcr.io/wlhe-cr/krun:main

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

RUN go build -a -gcflags='all=-N -l' -ldflags '-extldflags "-static"' -o /ws/hbgate ./cmd/gate

FROM ${BASE} AS istio

FROM ubuntu:bionic

COPY --from=build /ws/hbgate /usr/local/bin/hbgate

# Not using envoy
COPY --from=istio /ws/krun /ko-app/krun
COPY --from=istio /usr/local/bin/pilot-agent /usr/local/bin/

WORKDIR /

ENTRYPOINT ["/ko-app/krun"]
CMD ["/usr/local/bin/hbgate"]
