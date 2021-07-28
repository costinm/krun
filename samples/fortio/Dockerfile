# The base image is auto-build on github. You can also build your own base image from krun or
# your own fork. The image is based on Istio proxyv2 image, with krun added.
ARG BASE=ghcr.io/costinm/krun/krun:latest

# App code - can be an already built container, or the dockerfiel building.
FROM fortio/fortio:latest AS fortio

# Base is the Istio proxy image, including envoy, agent, krun
FROM ${BASE}

# Add the application and dependent files to the image
#COPY --from=fortio /usr/share/fortio /usr/share/fortio
COPY --from=fortio /usr/bin/fortio /usr/bin/fortio

# Make sure the entrypoint is set to krun. Krun is built with 'ko', which uses a specific dir.
ENTRYPOINT ["/ko-app/krun"]
# Normal application command
CMD ["/usr/bin/fortio", "server", "-http-port=8080"]
