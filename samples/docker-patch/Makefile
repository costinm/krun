# Local settings
ISTIOD?=$(shell cd ../..; pwd)
-include ${ISTIOD}/.local.mk

# GKE cluster used
PROJECT_ID=wlhe-cr
LOCATION=us-central1-c
CLUSTER=istio

# Region where CR will be deployed
REGION=us-central1

# Where to store the images
REPO=gcr.io/wlhe-cr

# The install assumes a subdomain is configured as a
# *.SUBDOMAIN A IP_ADDRESS
# where IP_ADDRES is the Istiod Ingress gateway address.
#
# In addition, CertManager/etc ACME certificates and routes are set for
# istiod.SUBDOMAIN
SUBDOMAIN=wlhe.i.webinf.info

#SVC=proxybase-dmgksid7iq-uc.a.cloud-run-qual.sandbox.google.com

SVC=fortio-dmgksid7iq-uc.a.cloud-run-qual.sandbox.google.com
######

# Suffix for the CR app
CR_SUFFIX=-icq63pqnqq-uc.a.run.app

ISTIOD_URL?=istiod.${SUBDOMAIN}:443

NS=fortio

all: push-fortio deploy-fortio

istio-fortio: patch-fortio deploy-istio-fortio

push-fortio: docker-fortio
	docker push ${REPO}/fortio:latest

# Local build - using ko.local for krun, local docker build
local/all: push-fortio-local deploy-fortio

push-fortio-local:
	(cd ../..; make local/krun)
	docker build . -f Dockerfile.fortio \
		--build-arg BASE=ko.local/krun:latest \
		-t ${REPO}/fortio:latest
	docker push ${REPO}/fortio:latest

# Build the image using the proxy as base
docker-fortio:
	docker pull ghcr.io/costinm/krun/krun:latest
	docker build . -f Dockerfile.fortio -t ${REPO}/fortio:latest

# WIP - use the app's base image and add the proxy
patch-fortio:
	docker pull ghcr.io/costinm/krun/krun:latest
	docker build . -f ../Dockerfile.istio-patch \
       --build-arg NS=fortio \
       --build-arg APP=cr-fortio \
       --build-arg BASE=ubuntu \
       -t ${REPO}/istio-fortio:latest

#       --build-arg BASE=costinm/fortio:latest \

push-istio-fortio: patch-fortio
	docker push ${REPO}/istio-fortio:latest

# GKE cluster used by cloudrun
K8S_CLUSTER_ARGS=--set-env-vars="PROJECT=${PROJECT_ID}" \
                 --set-env-vars="LOCATION=${LOCATION}" \
                 --set-env-vars="CLUSTER=${CLUSTER}" \
        		 --set-env-vars="XDS_ADDR=${ISTIOD_URL}"

# Enable the built-in sshd server, with cert auth
SSH_DEBUG_ARGS=--set-env-vars="SSH_CA=sshca.${SUBDOMAIN}:443" \
               --set-env-vars="SSH_AUTH=$(shell cat ~/.ssh/id_ecdsa.pub)" \

cr/fortio: push-fortio-local deploy-fortio

deploy-fortio:
	gcloud alpha run deploy fortio \
		  --platform managed --project ${PROJECT_ID} \
		  --region ${REGION} \
          --sandbox=minivm \
         --allow-unauthenticated \
         --use-http2 \
         --port 15009 \
         --min-instances=1 \
         --image ${REPO}/fortio:latest \
 		  ${K8S_CLUSTER_ARGS} \
 		  ${SSH_DEBUG_ARGS} \
         --set-env-vars="HTTP_PROXY=127.0.0.1:15080" \
         --set-env-vars="LABEL_FOO=bar2" \
         --set-env-vars="POD_NAMESPACE=fortio" \
 		 --set-env-vars="POD_NAME=fortio-cr" \
 		 --set-env-vars="LABEL_APP=fortio-cr" \
 		  --vpc-connector PROJECT_IDs/${PROJECT_ID}/locations/${REGION}/connectors/serverlesscon \

# Deploy the Fortio image with Istio patch
deploy-istio-fortio:
	gcloud alpha run deploy istio-fortio \
		  --platform managed --PROJECT_ID ${PROJECT_ID} \
		  --region ${REGION} \
          --sandbox=minivm \
         --allow-unauthenticated \
         --use-http2 \
         --port 15009 \
         --min-instances=1 \
         --image ${REPO}/istio-fortio:latest \
 		  ${K8S_CLUSTER_ARGS} \
 		  ${SSH_DEBUG_ARGS} \
 		  --vpc-connector PROJECT_IDs/${PROJECT_ID}/locations/${REGION}/connectors/serverlesscon \

# Deploy the same thing in asia
deploy-fortio-asia:
	gcloud alpha run deploy fortio-asia \
		  --platform managed --PROJECT_ID ${PROJECT_ID} \
		  --region asia-northeast1 \
		  --sandbox=minivm \
         --allow-unauthenticated \
 		  ${K8S_CLUSTER_ARGS} \
 		  ${SSH_DEBUG_ARGS} \
         --use-http2 \
         --port 15009 \
         --min-instances=1 \
         --image ${REPO}/fortio:latest \
         --set-env-vars="HTTP_PROXY=127.0.0.1:15080" \
         --set-env-vars="POD_NAMESPACE=fortio" \
 		 --set-env-vars="POD_NAME=fortio-cr" \
         --set-env-vars="LABEL_APP=fortio-cr" \

 		  #--vpc-connector projects/wlhe-cr/locations/us-central1/connectors/serverlesscon \

# SSH to the deployed CloudRun using HBONE
ssh-fortio:
	 ssh -v  -F /dev/null -o StrictHostKeyChecking=no \
        -o "UserKnownHostsFile /dev/null" \
 	    -o ProxyCommand='hbone https://fortio-icq63pqnqq-uc.a.run.app:443/_hbone/22' \
     	root@proxybase

# Test instance in asia
ssh-fortio-asia:
	 ssh -v  -F /dev/null -o StrictHostKeyChecking=no \
        -o "UserKnownHostsFile /dev/null" \
 	    -o ProxyCommand='hbone https://fortio-asia-icq63pqnqq-an.a.run.app:443/_hbone/22' \
     	root@proxybase

setupcon-sharedvpc:
	gcloud services enable vpcaccess.googleapis.com

	# Requires a /28 subnet - shared vpc
	gcloud compute networks vpc-access connectors create serverlesscon \
    --region asia-northeast1 \
    --subnet default \
    --subnet-project wlhe-cr\
    --min-instances 2 \
    --max-instances 10 \

	#
	gcloud compute networks vpc-access connectors create CONNECTOR_NAME \
	--network VPC_NETWORK \
	--region REGION \
	--range IP_RANGE

###################################################
# Running and testing in local docker
ADC=${HOME}/.config/gcloud/legacy_credentials/${USER}/adc.json
export ADC


# Local run for the fortio image.
run-fortio:
	docker run -it --rm \
		--name fortio \
        -p 9999:15009 \
        -e XDS_ADDR=istiod.wlhe.i.webinf.info:443 \
                -e CLUSTER=${CLUSTER} \
                -e PROJECT=${PROJECT_ID} \
                -e LOCATION=${LOCATION} \
                -e CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE=/var/run/secrets/google/google.json \
                -e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
                -v ${ADC}:/var/run/secrets/google/google.json:ro \
                ${REPO}/fortio:latest \
                /bin/bash

#        -e XDS_ADDR=istiod.wlhe.i.webinf.info:443 \
#               -e CLUSTER=${CLUSTER} \
#                -e PROJECT=${PROJECT_ID} \
#                -e LOCATION=${LOCATION} \
#                -e GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/google/google.json \
#


run-istio-fortio:
	docker run -it --rm \
		--name fortio \
        -p 9999:15009 \
        --entrypoint /bin/sh \
                -v ${ADC}:/var/run/secrets/google/google.json:ro \
                ${REPO}/istio-fortio:latest \


local-ssh-fortio:
	 ssh -F /dev/null -o "UserKnownHostsFile /dev/null" \
 		-o StrictHostKeyChecking=no \
 		-o ProxyCommand='hbone http://127.0.0.1:9999/_hbone/22'  \
     		root@proxybase

