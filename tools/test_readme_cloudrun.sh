#!/bin/bash

export PROJECT_ID=${PROJECT_ID:-wlhe-cr}
export CLUSTER_LOCATION=${CLUSTER_LOCATION:-us-central1-c}
export CLUSTER_NAME=${CLUSTER_NAME:-istio}

export REGION=${REGION:-us-central1}

export WORKLOAD_NAMESPACE=${WORKLOAD_NAMESPACE:-fortio}
export WORKLOAD_NAME=${WORKLOAD_NAME:-cloudrun}

export WORKLOAD_SERVICE_ACCOUNT=k8s-${WORKLOAD_NAMESPACE}@${PROJECT_ID}.iam.gserviceaccount.com

export CLOUDRUN_SERVICE=${WORKLOAD_NAME}

# ASM setup not covered - the project must have MCP or in-cluster installed.

# VPC connector not covered - must have a connector 'serverlesscon' created.

# Cluster setup not covered
# kubectl apply -f manifests/istio-system-discovery-rbac.yaml

gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}



# Once per namespace setup for the Google service account and permissions.
function setup_namespace() {

  gcloud --project ${PROJECT_ID} iam service-accounts create k8s-${WORKLOAD_NAMESPACE} \
        --display-name "Service account with access to ${WORKLOAD_NAMESPACE} k8s namespace"

  gcloud --project ${PROJECT_ID} projects add-iam-policy-binding \
              ${PROJECT_ID} \
              --member="serviceAccount:k8s-${WORKLOAD_NAMESPACE}@${PROJECT_ID}.iam.gserviceaccount.com" \
              --role="roles/container.clusterViewer"

  kubectl create ns ${WORKLOAD_NAMESPACE}
  kubectl label namespace ${WORKLOAD_NAMESPACE} istio-injection- istio.io/rev=asm-managed --overwrite

  cat manifests/rbac.yaml | envsubst | kubectl apply -f -
}

function deploy_app() {
  gcloud alpha run deploy ${CLOUDRUN_SERVICE} \
          --platform managed \
          --project ${PROJECT_ID} \
          --region ${REGION} \
          --execution-environment=gen2 \
          --service-account=k8s-${WORKLOAD_NAMESPACE}@${PROJECT_ID}.iam.gserviceaccount.com \
          --allow-unauthenticated \
          --use-http2 \
          --port 15009 \
          --image ${IMAGE} \
          --vpc-connector projects/${PROJECT_ID}/locations/${REGION}/connectors/serverlesscon \
         --set-env-vars="CLUSTER_NAME=${CLUSTER_NAME}" \
         --set-env-vars="CLUSTER_LOCATION=${CLUSTER_LOCATION}"
}

# Deploy the in-cluster test application
function setup_fortio() {
  kubectl apply -f https://raw.githubusercontent.com/costinm/krun/main/samples/fortio/in-cluster.yaml
}

