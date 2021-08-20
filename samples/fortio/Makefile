# Local settings - create a config file to avoid passing parms on each
# call.
-include .local.mk
-include ../.local.mk
-include ../../.local.mk

# Defaults, using the internal test cluster - must be overriden

# GKE cluster used
PROJECT_ID?=wlhe-cr
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

FORTIO_IMAGE?=${REPO}/fortio-cr:latest

ISTIOD_URL?=istiod.${SUBDOMAIN}:443

# Namespace to attach to.
NS=fortio

# Enable the built-in sshd server, with cert auth
SSH_DEBUG_ARGS=--set-env-vars="SSH_AUTH=$(shell cat ~/.ssh/id_ecdsa.pub)"

# If the SSH CA is installed, this will also provide server authentication, you can
# remove "-o StrictHostKeyChecking=no -o "UserKnownHostsFile /dev/null""
#--set-env-vars="SSH_CA=sshca.${SUBDOMAIN}:443" \


# Create fortio+proxy image, deploy to CloudRun
all: image push deploy

push: image
	docker push ${FORTIO_IMAGE}

# Build the image using the proxy as base
image:
	docker build . -t ${FORTIO_IMAGE}

# For gvisor: add
# --set-env-vars="HTTP_PROXY=127.0.0.1:15080"

#SANDBOX?=--sandbox=minivm
SANDBOX?=--execution-environment=gen2

SERVICE?=fortio-${CLUSTER_NAME}${SUFFIX}
# Deploy to cloudrun
# Adding XDS_ADDR=READ_DOMAIN:443 will skip looking up for MCP configmap and use that address instead.
deploy:
	gcloud alpha run deploy ${SERVICE} \
		  --platform managed --project ${PROJECT_ID} \
		  --region ${REGION} ${SANDBOX} \
		  --service-account=k8s-${NS}@${PROJECT_ID}.iam.gserviceaccount.com \
         --vpc-connector projects/${PROJECT_ID}/locations/${REGION}/connectors/serverlesscon \
         --allow-unauthenticated \
         --use-http2 \
         --port 15009 \
         --min-instances=1 \
         --image ${FORTIO_IMAGE} \
         --set-env-vars="CLUSTER_LOCATION=${CLUSTER_LOCATION}" \
         --set-env-vars="CLUSTER_NAME=${CLUSTER_NAME}" \
  		 --set-env-vars="LABEL_DEPLOY=$(shell date +%y%m%d-%H%M)" \
         ${SSH_DEBUG_ARGS} ${EXTRA} \
 		 --set-env-vars="DEPLOY=$(shell date +%y%m%d-%H%M)"

# Port 14009 for using envoy for ingress
# Port 8080 for going directly to the app
# Port 15009 for using KRun and 'native' hbone.

pull:
	docker pull ghcr.io/costinm/krun/krun:latest # Get latest

SVC?=https://fortio${CLUSTER_NAME}${SUFFIX}-icq63pqnqq-uc.a.run.app:443

# SSH to the deployed CloudRun using HBONE
# SUFFIX can be used for the other variants
ssh:
	 ssh -F /dev/null -v \
        -o StrictHostKeyChecking=no -o "UserKnownHostsFile /dev/null" \
 	    -o ProxyCommand='hbone ${SVC}/_hbone/22' \
     	root@proxybase ${SSH_ARGS}

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
	cat ../../manifests/rbac.yaml | NS=${NS} envsubst  | kubectl apply -f -

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

