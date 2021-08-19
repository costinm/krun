# Specialized builder - should include all deps.

FROM golang:alpine AS go

FROM gcr.io/google.com/cloudsdktool/cloud-sdk:alpine

RUN gcloud components install -q alpha

RUN apk update && apk add gcc bash musl-dev openssl-dev ca-certificates make && update-ca-certificates

COPY --from=go /usr/local/go /usr/local/

ENV PATH=$PATH:/usr/local/go/bin

ENTRYPOINT ["/bin/sh"]
