# Local settings - create a config file to avoid passing parms on each
# call.
-include .local.mk
-include ../.local.mk
-include ../../.local.mk

# Defaults, using the internal test cluster - must be overriden

# GKE cluster used
PROJECT_ID?=wlhe-cr
PROJECT_NUMBER?=$(shell gcloud projects describe ${PROJECT_ID} --format="value(projectNumber)")
CLUSTER_LOCATION?=us-central1-c
CLUSTER_NAME?=asm-cr

# Region where CR will be deployed
REGION?=us-central1

# The install assumes a subdomain is configured as a
# *.SUBDOMAIN A IP_ADDRESS
# where IP_ADDRES is the Istiod Ingress gateway address.
#
# In addition, CertManager/etc ACME certificates and routes are set for
# istiod.SUBDOMAIN
SUBDOMAIN=wlhe.i.webinf.info

# Suffix for the CR app - this is used for the 'ssh' debug commands, must
# be set after the first deploy.
# TODO: add a small cli to list the CR apps and get the suffix
CR_SUFFIX=-icq63pqnqq-uc.a.run.app

################ Values derived from .local.mk

# Where to store the images
REPO=gcr.io/${PROJECT_ID}

IMAGE?=${REPO}/fortio-cr:latest

ISTIOD_URL?=istiod.${SUBDOMAIN}:443

# Namespace to attach to.
NS=fortio

# GKE cluster used by cloudrun
K8S_CLUSTER_ARGS=--set-env-vars="PROJECT=${PROJECT_ID}" \
                 --set-env-vars="LOCATION=${CLUSTER_LOCATION}" \
                 --set-env-vars="CLUSTER=${CLUSTER_NAME}" \
        		 --set-env-vars="XDS_ADDR=${ISTIOD_URL}"

# Enable the built-in sshd server, with cert auth
SSH_DEBUG_ARGS=--set-env-vars="SSH_AUTH=$(shell cat ~/.ssh/id_ecdsa.pub)"

# If the SSH CA is installed, this will also provide server authentication, you can
# remove "-o StrictHostKeyChecking=no -o "UserKnownHostsFile /dev/null""
#--set-env-vars="SSH_CA=sshca.${SUBDOMAIN}:443" \


# Create fortio+proxy image, deploy to CloudRun
all: image push deploy

push: image
	docker push ${IMAGE}

# Build the image using the proxy as base
image:
	docker build . -t ${IMAGE}

# For gvisor: add
# --set-env-vars="HTTP_PROXY=127.0.0.1:15080"

SANDBOX?=--sandbox=minivm

# Deploy to cloudrun
deploy:
	gcloud alpha run deploy fortio-${CLUSTER_NAME}${SUFFIX} \
		  --platform managed --project ${PROJECT_ID} \
		  --region ${REGION} ${SANDBOX} \
         --vpc-connector projects/${PROJECT_ID}/locations/${REGION}/connectors/serverlesscon \
         --allow-unauthenticated \
         --use-http2 \
         --port 15009 \
         --min-instances=1 \
         --image ${IMAGE} \
         --set-env-vars="CLUSTER_LOCATION=${CLUSTER_LOCATION}" \
         --set-env-vars="CLUSTER_NAME=${CLUSTER_NAME}" \
         --set-env-vars="XDS_ADDR=staging-meshconfig.sandbox.googleapis.com:443" \
         --set-env-vars="ISTIO_META_CLOUDRUN_ADDR=asm-stg-asm-cr-asm-managed-rapid-c-2o26nc3aha-uc.a.run.app:443" \
          ${SSH_DEBUG_ARGS} ${EXTRA} \
         --set-env-vars="WORKLOAD_NAMESPACE=fortio" \
 		 --set-env-vars="WORKLOAD_NAME=fortio-cr" \
 		 --set-env-vars="LABEL_APP=fortio-cr" \
 		 --set-env-vars="DEPLOY=$(shell date +%y%m%d-%H%M)"

# WIP - will need additional setup for exposing Istiod on a ILB or gateway with certs.
deploy/oss:
	gcloud alpha run deploy fortio-${CLUSTER_NAME}${SUFFIX} \
		  --platform managed --project ${PROJECT_ID} \
		  --region ${REGION} ${SANDBOX} \
         --allow-unauthenticated \
         --use-http2 \
         --port 15009 \
         --min-instances=1 \
         --image ${IMAGE} \
 		  ${K8S_CLUSTER_ARGS} \
 		  ${SSH_DEBUG_ARGS} ${EXTRA} \
          --set-env-vars="HTTP_PROXY=127.0.0.1:15080" \
         --set-env-vars="WORKLOAD_NAMESPACE=fortio" \
 		 --set-env-vars="WORKLOAD_NAME=fortio-cr" \
 		 --set-env-vars="LABEL_APP=fortio-cr" \
 		 --set-env-vars="DEPLOY=$(shell date +%y%m%d-%H%M)" \
 		  --vpc-connector projects/${PROJECT_ID}/locations/${REGION}/connectors/serverlesscon \


# WIP - using gvisor and whitebox
deploy-gvisor:
	$(MAKE) deploy SUFFIX=-gvisor SANDBOX=""


pull:
	docker pull ghcr.io/costinm/krun/krun:latest # Get latest

# SSH to the deployed CloudRun using HBONE
# SUFFIX can be used for the other variants
ssh:
	 ssh -F /dev/null \
        -o StrictHostKeyChecking=no -o "UserKnownHostsFile /dev/null" \
 	    -o ProxyCommand='hbone https://fortio${SUFFIX}-icq63pqnqq-uc.a.run.app:443/_hbone/22' \
     	root@proxybase

config_dump:
	 ssh -F /dev/null -o StrictHostKeyChecking=no \
        -o "UserKnownHostsFile /dev/null" \
 	    -o ProxyCommand='hbone https://fortio${SUFFIX}-icq63pqnqq-uc.a.run.app:443/_hbone/22' -- sh curl localhost:15000/config_dump

###########################################

# Run first, to create the permissions
setup: setup-gsa setup-rbac

setup-gsa:
	gcloud --project ${PROJECT_ID} iam service-accounts create k8s-${NS} \
      --display-name "Service account with access to ${NS} k8s namespace"
	gcloud --project ${PROJECT_ID} projects add-iam-policy-binding \
            ${PROJECT_ID} \
            --member="serviceAccount:k8s-${NS}@${PROJECT_ID}.iam.gserviceaccount.com" \
            --role="roles/container.clusterViewer"

setup-rbac:
	cat rbac.yaml | NS=${NS} envsubst  | kubectl apply -f -

logs-project:
	gcloud logging read 'resource.type = "project" OR resource.type = "cloud_run_revision"'

# textPayload:SyncAddress --limit=50 --format=json
logs:
	#gcloud logging read 'resource.type="cloud_run_revision" AND resource.labels.location = "us-central1" AND resource.labels.service_name="fortio${SUFFIX}"'
	gcloud --project ${PROJECT_ID} logging read \
		--format "csv(resource.labels.service_name,textPayload)" \
		--freshness 1h \
 		'resource.type="cloud_run_revision" AND resource.labels.location = "us-central1" AND resource.labels.service_name="fortio${SUFFIX}"'

setupcon-sharedvpc:
	gcloud services enable vpcaccess.googleapis.com
	gcloud compute networks vpc-access connectors create serverlesscon \
    --region ${REGION} \
    --subnet default \
    --subnet-project ${PROJECT_ID} \
    --min-instances 2 \
    --max-instances 10 \


###################################################
# Running and testing in local docker
GSA?=${USER}
ADC=${HOME}/.config/gcloud/legacy_credentials/${GSA}/adc.json
export ADC

# Local build - using ko.local for krun, local docker build
run: image run/docker


# Run the image in a local docker container, using a service account file
run/docker:
	docker run -it --rm \
		--name fortio \
        -p 9999:15009 \
        -e XDS_ADDR=istiod.wlhe.i.webinf.info:443 \
		-e CLUSTER_NAME=${CLUSTER_NAME} \
		-e PROJECT_ID=${PROJECT_ID} \
		-e CLUSTER_LOCATION=${CLUSTER_LOCATION} \
		-e WORKLOAD_NAMESPACE=fortio \
		-e WORKLOAD_NAME=fortio-cr _NAME\
		-e LABEL_APP=fortio-cr \
		-e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
		-v ${ADC}:/var/run/secrets/google/google.json:ro \
		${IMAGE} \
		/bin/bash


run/docker-mcp:
	docker run -it --rm \
		--name fortio \
        -p 9999:15009 \
		-e CLUSTER_NAME=${CLUSTER_NAME} \
		-e PROJECT_ID=${PROJECT_ID} \
		-e PROJECT_NUMBER=${PROJECT_NUMBER} \
        -e CLUSTER_LOCATION=${CLUSTER_LOCATION} \
        -e XDS_ADDR=staging-meshconfig.sandbox.googleapis.com:443 \
        -e ISTIO_META_CLOUDRUN_ADDR=asm-stg-asm-cr-asm-managed-rapid-c-2o26nc3aha-uc.a.run.app:443 \
         \
		-e WORKLOAD_NAMESPACE=fortio \
		-e WORKLOAD_NAME=fortio-cr \
		-e LABEL_APP=fortio-cr \
		\
		-e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
		-v ${ADC}:/var/run/secrets/google/google.json:ro \
		${IMAGE} \
		/bin/bash

# Will run 'krun' - to debug krun use
#        --entrypoint /bin/sh


# SSH into the local docker (for testing ssh - docker exec is more convenient )
ssh/docker:
	 ssh -F /dev/null -o "UserKnownHostsFile /dev/null" \
 		-o StrictHostKeyChecking=no \
 		-o ProxyCommand='hbone http://127.0.0.1:9999/_hbone/22'  \
     		root@proxybase

# Deploy the same thing in asia
deploy/asia:
	$(MAKE) deploy SUFFIX=asia REGION=asia-northeast1

