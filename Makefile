
# Must define:
# CLUSTER
# PROJECT_ID
# LOCATION
# SUBDOMAIN - for now we require Istiod to use an ACME cert and proper domain ('external istiod' style)
# USER - User logged in gcloud, used to find adc for local tests
-include .local.mk

PROJECT_ID?=wlhe-cr

ISTIO_CHARTS?=../istio/manifests/charts
REV?=v1-11

# Github actions use this.
KO_DOCKER_REPO?=gcr.io/${PROJECT_ID}/krun
export KO_DOCKER_REPO

# For testing/dev in local docker
ADC?=${HOME}/.config/gcloud/legacy_credentials/${USER}/adc.json
export ADC

KRUN_IMAGE?=${KO_DOCKER_REPO}:latest

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
OUT?=${ROOT_DIR}/../out/krun

TAG?=latest

# Push krun - the github action on push will do the same.
# This is the fastest way to push krun - permission required to KO_DOCKER_REPO
push/krun:
	ko publish -B ./

# Build and tag krun image locally, will be used in the next phase and for
# local testing.
build: build/krun

images: build
	(cd samples/fortio; make image)

build/krun:
	KO_IMAGE=$(shell ko publish -L -B ./) $(MAKE) docker/tag

build/docker:
	docker build . -t ${KRUN_IMAGE}

build/docker-hgate:
	docker build . -t gcr.io/${PROJECT_ID}/hbgate:${TAG}


docker/tag:
	docker tag ${KO_IMAGE} ko.local/krun:latest && \
	docker tag ${KO_IMAGE} ${KRUN_IMAGE}

################# Testing / local dev #################

# Run krun in a docker image, get a shell. Will use MCP.
docker/run-noxds:
	docker run -it --rm \
		-e CLUSTER_NAME=${CLUSTER_NAME} \
		-e PROJECT_ID=${PROJECT_ID} \
		-e CLUSTER_LOCATION=${CLUSTER_LOCATION} \
		-e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
		-v ${ADC}:/var/run/secrets/google/google.json:ro \
		${KRUN_IMAGE} \
	   /bin/bash

# Run in local docker, using ADC for auth and an explicit XDS address
docker/run-xds-adc:
	docker run -it --rm \
		-e XDS_ADDR=istiod.wlhe.i.webinf.info:443 \
		-e CLUSTER_NAME=${CLUSTER_NAME} \
		-e PROJECT_ID=${PROJECT_ID} \
		-e CLUSTER_LOCATION=${CLUSTER_LOCATION} \
		-e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
		-v ${ADC}:/var/run/secrets/google/google.json:ro \
		${KRUN_IMAGE} \
		/bin/bash


# Run hbgate in a local docker container, for testing.
# Will connect to the cluster.
#
# ISTIO_META_INTERCEPTION_MODE disable interception (not using it).
# DISABLE_ENVOY also disables envoy - only using the cert part in istio-agent
docker/run-hbgate:
	docker run -it --rm \
		-e CLUSTER_NAME=${CLUSTER_NAME} \
		-e PROJECT_ID=${PROJECT_ID} \
		-e CLUSTER_LOCATION=${CLUSTER_LOCATION} \
		-e DISABLE_ENVOY=true \
		-e ISTIO_META_INTERCEPTION_MODE=NONE \
		-e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
		-p 15441:15441 \
		-v ${ADC}:/var/run/secrets/google/google.json:ro \
		 gcr.io/${PROJECT_ID}/hbgate:${TAG} \
	   /bin/bash

local/run-kubeconfig:
	docker run  -e KUBECONFIG=/var/run/kubeconfig -v ${HOME}/.kube/config:/var/run/kubeconfig:ro -it  \
		${KRUN_IMAGE}  /bin/bash

push/fortio:
	(cd samples/fortio; make image push)

all: images push/fortio deploy/fortio

deploy/fortio:
	(cd samples/fortio; make deploy)

## Cluster setup for samples and testing
deploy/k8s-fortio:
	helm upgrade --install \
		-n fortio \
		fortio \
 		samples/charts/fortio

template/k8s-fortio:
	helm template \
		-n fortio \
		fortio \
 		samples/charts/fortio > samples/fortio/in-cluster.yaml

# Update base images, for build/krun ( local build )
pull:
	# Custom build
	#docker pull gcr.io/wlhe-cr/proxyv2:cloudrun
	docker pull gcr.io/istio-testing/proxyv2:latest

# Get deps
deps:
	curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
	chmod +x kubectl
	# TODO: helm, ko


test:
	go test -timeout 2m -v ./...

# Canary will deploy a 'canary' version of a cloudrun instance using the current golden image, and verify it works
canary: canary/deploy canary/test

canary/deploy:
	(cd samples/fortio; REGION=us-central1 CLUSTER_NAME=asm-cr CLUSTER_LOCATION=us-central1-c \
    	make deploy)
    # OSS/ASM with Istiod exposed in Gateway, with ACME certs
	(cd samples/fortio; REGION=us-central1 CLUSTER_NAME=istio CLUSTER_LOCATION=us-central1-c \
		EXTRA="--set-env-vars XDS_ADDR=istiod.wlhe.i.webinf.info:443" \
		make deploy)

canary/test:
	curl  -v  https://fortio-asm-cr-icq63pqnqq-uc.a.run.app/fortio/fetch2/?url=http%3A%2F%2Ffortio.fortio.svc%3A8080%2Fecho
	curl  -v https://fortio-istio-icq63pqnqq-uc.a.run.app/fortio/fetch2/?url=http%3A%2F%2Ffortio.fortio.svc%3A8080%2Fecho
	curl -v  https://fortio-istio-icq63pqnqq-uc.a.run.app/fortio/fetch2/?url=http%3A%2F%2Flocalhost%3A15000%2Fconfig_dump

# A single version of Istiod - using a version-based revision name.
# The version will be associated with labels using in the other charts.
deploy/istiod:
	# Install istiod.
	# Telemetry configs can be installed as a separate chart - this
	# avoids upgrade issues for 1.4 skip-version.
	# TODO: add telementry to docker image
	helm upgrade --install \
 		-n istio-system \
 		istiod-${REV} \
        ${ISTIO_CHARTS}/istio-control/istio-discovery \
		--set revision=${REV} \
		--set telemetry.enabled=true \
		--set meshConfig.trustDomain="${PROJECT_ID}.svc.id.goog" \
		--set global.sds.token.aud="${PROJECT_ID}.svc.id.goog" \
		--set pilot.env.TOKEN_AUDIENCES="${PROJECT_ID}.svc.id.goog\,istio-ca" \
		--set meshConfig.proxyHttpPort=15080 \
        --set meshConfig.accessLogFile=/dev/stdout \
        --set pilot.replicaCount=1 \
        --set pilot.autoscaleEnabled=false \
		--set pilot.env.PILOT_ENABLE_WORKLOAD_ENTRY_AUTOREGISTRATION=true \
		--set pilot.env.PILOT_ENABLE_WORKLOAD_ENTRY_HEALTHCHECKS=true

# Whitebox:
# Istio install:
# 		--set meshConfig.proxyHttpPort=15080 \
# Cloudrun:
#  --set-env-vars="HTTP_PROXY=127.0.0.1:15080"

# Create the builder docker image, used in GCB
gcb/builder:
	gcloud builds --project ${PROJECT_ID} submit . --config=tools/gcb/cloudbuild.yaml

gcb/builder-gcloud:
	gcloud builds --project ${PROJECT_ID} submit . --config=tools/gcloud-alpha/cloudbuild.yaml

gcb/builder-ko:
	gcloud builds --project ${PROJECT_ID} submit . --config=tools/ko/cloudbuild.yaml

# Local testing using CI/CD. This uses the 'ko' variant - since kaniko doesn't work locally (and is fastest on GCB)
gcb/local:
	mkdir -p ${OUT}/gcb-local
	cloud-build-local --dryrun=false --push=true --write-workspace=${OUT}/gcb-local  --substitutions=BRANCH_NAME=local,COMMIT_SHA=local --config=tools/local/cloudbuild.yaml .

gcb/build-hgate:
	gcloud builds --project ${PROJECT_ID} submit  --config=cmd/gate/cloudbuild.yaml .

gcb/build:
	gcloud builds --project ${PROJECT_ID} submit .

