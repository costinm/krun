
# Must define:
# CLUSTER
# PROJECT
# LOCATION
# USER - Used logged into gcloud, to find adc
-include .local.mk

#IMAGE=ghcr.io/costinm/krun/krun:latest
IMAGE=ko.local/krun:latest


ADC=${HOME}/.config/gcloud/legacy_credentials/${USER}/adc.json
export ADC

local/run-noxds:
	docker run -it --rm \
		-e CLUSTER=${CLUSTER} \
		-e PROJECT=${PROJECT} \
		-e LOCATION=${LOCATION} \
		-e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
		-v ${ADC}:/var/run/secrets/google/google.json:ro \
		${IMAGE} \
	   /bin/bash


local/run-xds:
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

build/docker-local:
	KO_DOCKER_REPO=ko.local ko publish -B .

## In-cluster

deploy/fortio-mcp:
	helm upgrade --install \
		-n fortio-mcp \
		fortio-mcp \
 		samples/charts/fortio

deploy/fortio:
	helm upgrade --install \
		-n fortio \
		fortio \
 		samples/charts/fortio

# Push krun to ghcr.io - the actions will do the same
push/krun:
	KO_DOCKER_REPO=ghcr.io/costinm/krun/krun ko publish -B ./

# Build krun image locally.
local/krun:
	#docker pull gcr.io/wlhe-cr/proxyv2:cloudrun
	#docker pull gcr.io/istio-testing/proxyv2:latest
	KO_DOCKER_REPO=ko.local ko publish -B ./
