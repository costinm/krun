# Local settings - create a config file to avoid passing parms on each
# call.
-include .local.mk
-include ../../.local.mk

# Defaults, using the internal test cluster - must be overriden

# GKE cluster used
PROJECT_ID?=wlhe-cr
CLUSTER_LOCATION?=us-central1-c
CLUSTER_NAME?=istio

# Region where CR will be deployed
REGION?=us-central1


################ derived values

# Where to store the images
REPO?=gcr.io/${PROJECT_ID}

# Base image, including istio-proxy, envoy, starter. Built by the CI/CD on the test project.
GOLDEN_IMAGE?=gcr.io/wlhe-cr/krun:main

FORTIO_IMAGE?=${REPO}/fortio-cr:latest

# Namespace to attach to.
WORKLOAD_NAMESPACE?=fortio

WORKLOAD_SERVICE_ACCOUNT=k8s-${WORKLOAD_NAMESPACE}@${PROJECT_ID}.iam.gserviceaccount.com

export WORKLOAD_SERVICE_ACCOUNT
export WORKLOAD_NAMESPACE


# Name of the workload. For CloudRun, this is also the default 'canonical service' and the name of the associated
# service entry/service.
WORKLOAD_NAME?=fortio-cr
export WORKLOAD_NAME

# Create fortio+proxy image, deploy to CloudRun
all: image push deploy

# Build the image using the proxy as base
image:
	docker build --build-arg=BASE=${GOLDEN_IMAGE} -t ${FORTIO_IMAGE} ${DOCKER_BUILD_ARGS} .

push:
	docker push ${FORTIO_IMAGE}


SERVICE?=${WORKLOAD_NAME}

# Enable the built-in sshd server, with cert auth. Temp, will move to a config map in cluster
SSH_DEBUG_ARGS=--set-env-vars="SSH_AUTH=$(shell cat ~/.ssh/id_ecdsa.pub)"

# Deploy to cloudrun
# Adding XDS_ADDR=READ_DOMAIN:443 will skip looking up for MCP configmap and use that address instead.
deploy:
	gcloud alpha run deploy ${SERVICE} \
		  --execution-environment=gen2 \
		  --platform managed --project ${PROJECT_ID} --region ${REGION} \
		  --service-account=${WORKLOAD_SERVICE_ACCOUNT} \
          --vpc-connector projects/${PROJECT_ID}/locations/${REGION}/connectors/serverlesscon \
         \
         --allow-unauthenticated \
         \
         --use-http2 \
         --port 15009 \
         \
         --image ${FORTIO_IMAGE} \
         --set-env-vars="CLUSTER_LOCATION=${CLUSTER_LOCATION}" \
         --set-env-vars="CLUSTER_NAME=${CLUSTER_NAME}" \
  		 ${SSH_DEBUG_ARGS} ${EXTRA} \
 		 --set-env-vars="DEPLOY=$(shell date +%y%m%d-%H%M)"

# Port 14009 for using envoy for ingress
# Port 8080 for going directly to the app
# Port 15009 for using KRun and 'native' hbone.

pull:
	docker pull ghcr.io/costinm/krun/krun:latest # Get latest

# SSH to the deployed CloudRun using HBONE
ssh: CR_URL=$(shell gcloud run services describe ${SERVICE} --format="value(status.address.url)")
ssh:
	 ssh -F /dev/null -v \
        -o StrictHostKeyChecking=no -o "UserKnownHostsFile /dev/null" \
 	    -o ProxyCommand='hbone ${CR_URL}:443/_hbone/22' \
     	root@proxybase ${SSH_ARGS}

config_dump: CR_URL=$(shell gcloud run services describe ${SERVICE} --format="value(status.address.url)")
config_dump:
	 ssh -F /dev/null -o StrictHostKeyChecking=no \
        -o "UserKnownHostsFile /dev/null" \
 	    -o ProxyCommand='hbone ${CR_URL}/_hbone/22' -- sh curl localhost:15000/config_dump

###########################################

# Run first, to create the permissions
setup: setup-gsa setup-rbac

setup-gsa:
	gcloud --project ${PROJECT_ID} iam service-accounts create k8s-${WORKLOAD_NAMESPACE} \
      --display-name "Service account with access to ${WORKLOAD_NAMESPACE} k8s namespace"
	gcloud --project ${PROJECT_ID} projects add-iam-policy-binding \
            ${PROJECT_ID} \
            --member="serviceAccount:k8s-${WORKLOAD_NAMESPACE}@${PROJECT_ID}.iam.gserviceaccount.com" \
            --role="roles/container.clusterViewer"

setup-rbac:
	cat ../../manifests/rbac.yaml | envsubst  | kubectl apply -f -

logs-project:
	gcloud logging read 'resource.type = "project" OR resource.type = "cloud_run_revision"'

# textPayload:SyncAddress --limit=50 --format=json
logs:
	#gcloud logging read 'resource.type="cloud_run_revision" AND resource.labels.location = "us-central1" AND resource.labels.service_name="fortio${SUFFIX}"'
	gcloud --project ${PROJECT_ID} logging read \
		--format "csv(resource.labels.service_name,textPayload)" \
		--freshness 1h \
 		'resource.type="cloud_run_revision" AND resource.labels.location = "us-central1" AND resource.labels.service_name="${SERVICE}"'

setupcon-sharedvpc:
	gcloud services enable vpcaccess.googleapis.com
	gcloud compute networks vpc-access connectors create serverlesscon \
    --region ${REGION} \
    --subnet default \
    --subnet-project ${PROJECT_ID} \
    --min-instances 2 \
    --max-instances 10 \

