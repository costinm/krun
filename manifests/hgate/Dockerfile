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
RUN go mod download

COPY *.go ./
COPY cmd ./cmd/
COPY pkg ./pkg/
COPY kodata ./kodata/

RUN go build -a -gcflags='all=-N -l' -ldflags '-extldflags "-static"' -o /ws/krun ./


FROM ${BASE} AS istio

# Similar with the 'ko' runtime layout
COPY --from=build /ws/krun /ko-app/krun

COPY kodata/* /var/run/ko/

RUN  chown -R 1337 /var/run/ko

ENV KO_DATA_PATH=/var/run/ko

WORKDIR /

ENTRYPOINT ["/ko-app/krun"]
