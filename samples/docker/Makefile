# Local settings - create a config file to avoid passing parms on each
# call.
-include .local.mk
-include ../../.local.mk

# Defaults, using the internal test cluster - must be overriden

PROJECT_ID?=wlhe-cr
CLUSTER_LOCATION?=us-central1-c
CLUSTER_NAME?=asm-cr

################ Values derived from .local.mk

# Where to store the images
REPO=gcr.io/${PROJECT_ID}
IMAGE?=${REPO}/fortio-cr:latest

# Namespace to attach to.
NS=fortio

# Enable the built-in sshd server, with cert auth
SSH_DEBUG_ARGS=--set-env-vars="SSH_AUTH=$(shell cat ~/.ssh/id_ecdsa.pub)"

# Running and testing in local docker
GSA?=${USER}
ADC?=${HOME}/.config/gcloud/legacy_credentials/${GSA}/adc.json
export ADC

run:
	docker run -it --rm \
		--name fortio \
        -p 9999:14009 \
		-e PROJECT_ID=${PROJECT_ID} \
		-e PROJECT_NUMBER=$(shell gcloud projects describe ${PROJECT_ID} --format="value(projectNumber)") \
		-e CLUSTER_NAME=${CLUSTER_NAME} \
        -e CLUSTER_LOCATION=${CLUSTER_LOCATION} \
		-e WORKLOAD_NAMESPACE=fortio \
		-e WORKLOAD_NAME=cloudrun \
		\
		-e SSH_AUTH="$(shell cat ~/.ssh/id_ecdsa.pub)" \
		-e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
		-v ${ADC}:/var/run/secrets/google/google.json:ro \
		${IMAGE} \
		/bin/bash

# Other parameters:
#

# SSH into the local docker (for testing ssh - docker exec is more convenient )
ssh:
	 ssh -F /dev/null -o "UserKnownHostsFile /dev/null" \
 		-o StrictHostKeyChecking=no \
 		-o ProxyCommand='hbone http://127.0.0.1:9999/_hbone/22'  \
     		root@proxybase
