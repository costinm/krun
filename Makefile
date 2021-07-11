
# Must define:
# CLUSTER
# PROJECT
# LOCATION
# SUBDOMAIN - for now we require Istiod to use an ACME cert and proper domain ('external istiod' style)
# USER - User logged in gcloud, used to find adc for local tests
-include .local.mk

KO_DOCKER_REPO?=ghcr.io/costinm/krun/krun
export KO_DOCKER_REPO

ADC?=${HOME}/.config/gcloud/legacy_credentials/${USER}/adc.json
export ADC

IMAGE=ghcr.io/costinm/krun/krun:latest
#IMAGE?=ko.local/krun:latest

# Push krun - the github action on push will do the same
push/krun:
	ko publish -B ./

# Update base images, for build/krun ( local build )
pull:
	# Custom build
	docker pull gcr.io/wlhe-cr/proxyv2:cloudrun
	#docker pull gcr.io/istio-testing/proxyv2:latest


# Build and tag krun image locally.
build/krun:	KO_IMAGE=$(shell ko publish -L -B ./)
build/krun:
	docker tag ${KO_IMAGE} ko.local/krun:latest
	docker tag ${KO_IMAGE} ${IMAGE}


# Run krun in a docker image, get a shell - no pilot agent or envoy sidecar, since
# XDS_ADDR is not set.
docker/run-noxds:
	docker run -it --rm \
		-e CLUSTER=${CLUSTER} \
		-e PROJECT=${PROJECT} \
		-e LOCATION=${LOCATION} \
		-e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
		-v ${ADC}:/var/run/secrets/google/google.json:ro \
		${IMAGE} \
	   /bin/bash

# Run in local docker, using ADC for auth
docker/run-xds-adc:
	docker run -it --rm \
		-e XDS_ADDR=istiod.wlhe.i.webinf.info:443 \
		-e CLUSTER=${CLUSTER} \
		-e PROJECT=${PROJECT} \
		-e LOCATION=${LOCATION} \
		-e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
		-v ${ADC}:/var/run/secrets/google/google.json:ro \
		${IMAGE} \
		/bin/bash

local/run-xds-local:
	IMAGE=ko.local/krun:latest $(MAKE) local/run-xds


fortio/all: build/krun
	(cd samples/fortio; make push deploy)

ksa:
	(cd samples/fortio; make deploy SUFFIX=2 EXTRA=--service-account=fortio-default)

## Cluster setup for samples and testing

deploy/k8s-fortio:
	helm upgrade --install \
		-n fortio \
		fortio \
 		samples/charts/fortio

