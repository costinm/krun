
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
		-e CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=/var/run/secrets/google/google.json \
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
		-e CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=/var/run/secrets/google/google.json \
		-e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
		-v ${ADC}:/var/run/secrets/google/google.json:ro \
		${IMAGE} \
		/bin/bash

local/run-xds-local:
	IMAGE=ko.local/krun:latest $(MAKE) local/run-xds

build/docker-local:
	KO_DOCKER_REPO=ko.local ko publish -B .
