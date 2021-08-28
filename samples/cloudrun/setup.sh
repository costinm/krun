#!/bin/bash

export PROJECT_ID=${PROJECT_ID:-wlhe-cr}
export CLUSTER_LOCATION=${CLUSTER_LOCATION:-us-central1-c}
export CLUSTER_NAME=${CLUSTER_NAME:-asm-cr}
export REGION=${REGION:-us-central1}
export WORKLOAD_NAMESPACE=fortio 
export WORKLOAD_NAME=cloudrun
export WORKLOAD_SERVICE_ACCOUNT=k8s-${WORKLOAD_NAMESPACE}@${PROJECT_ID}.iam.gserviceaccount.com
export CLOUDRUN_SERVICE=${WORKLOAD_NAME}

# wlhe@: golden image needs to push to public repo
export  GOLDEN_IMAGE=gcr.io/wlhe-cr/krun:main
# Target image 
export  IMAGE=gcr.io/${PROJECT_ID}/fortio-cr:main

# Example command to create a regular cluster.
function create_cluster() {
  gcloud beta container --project "${PROJECT_ID}" clusters create \
    "${CLUSTER_NAME}" --zone "${CLUSTER_LOCATION}" \
    --no-enable-basic-auth \
    --cluster-version "1.21.3-gke.2000" \
    --release-channel "rapid" \
    --machine-type "e2-standard-8" \
    --image-type "COS_CONTAINERD" \
    --disk-type "pd-standard" \
    --disk-size "100" \
    --metadata disable-legacy-endpoints=true \
    --scopes "https://www.googleapis.com/auth/devstorage.read_only","https://www.googleapis.com/auth/logging.write","https://www.googleapis.com/auth/monitoring","https://www.googleapis.com/auth/servicecontrol","https://www.googleapis.com/auth/service.management.readonly","https://www.googleapis.com/auth/trace.append" \
    --max-pods-per-node "110" \
    --num-nodes "1" \
    --enable-stackdriver-kubernetes \
    --enable-ip-alias \
    --network "projects/${PROJECT_ID}/global/networks/default" \
    --subnetwork "projects/${PROJECT_ID}/regions/${REGION}/subnetworks/default" \
    --no-enable-intra-node-visibility \
    --default-max-pods-per-node "110" \
    --enable-autoscaling \
    --min-nodes "0" \
    --max-nodes "9" \
    --enable-network-policy \
    --no-enable-master-authorized-networks \
    --addons HorizontalPodAutoscaling,HttpLoadBalancing,GcePersistentDiskCsiDriver \
    --enable-autoupgrade \
    --enable-autorepair \
    --max-surge-upgrade 1 \
    --max-unavailable-upgrade 0 \
    --workload-pool "${PROJECT_ID}.svc.id.goog" \
    --enable-shielded-nodes \
    --node-locations "${CLUSTER_LOCATION}"

}

# WIP: using an autopilot cluster for configurations. Note that only gateways can run right now inside the
# autopilot - other workloads should be in regular clusters (iptables)
function create_cluster_autopilot() {
  gcloud beta container --project "${PROJECT_ID}" clusters create-auto \
    "${CLUSTER_NAME}" --region "${CLUSTER_LOCATION}" \
    --release-channel "regular" \
    --network "projects/${PROJECT_ID}/global/networks/default" \
    --subnetwork "projects/${PROJECT_ID}/regions/${REGION}/subnetworks/default" \
    --cluster-ipv4-cidr "/17" \
    --services-ipv4-cidr "/22"

  gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}
  kubectl create ns istio-system

}

function setup_asm() {
  curl https://storage.googleapis.com/csm-artifacts/asm/install_asm_1.10 > install_asm
  chmod +x install_asm

  # Managed CP:
  ./install_asm --mode install --output_dir ${CLUSTER_NAME} --enable_all --managed
}

# Per project setup
function setup_project() {
  	gcloud services enable --project ${PROJECT_ID} vpcaccess.googleapis.com
  	gcloud compute networks vpc-access connectors create serverlesscon \
      --project ${PROJECT_ID} \
      --region ${REGION} \
      --subnet default \
      --subnet-project ${PROJECT_ID} \
      --min-instances 2 \
      --max-instances 10
}


function setup_service_account() {
  gcloud --project ${PROJECT_ID} iam service-accounts create k8s-${WORKLOAD_NAMESPACE} \
        --display-name "Service account with access to ${WORKLOAD_NAMESPACE} k8s namespace"

  gcloud --project ${PROJECT_ID} projects add-iam-policy-binding \
              ${PROJECT_ID} \
              --member="serviceAccount:k8s-${WORKLOAD_NAMESPACE}@${PROJECT_ID}.iam.gserviceaccount.com" \
              --role="roles/container.clusterViewer"
}

function setup_namespace() {
  gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}

  kubectl create ns fortio
  kubectl label namespace fortio istio-injection- istio.io/rev=asm-managed-rapid --overwrite
  cat manifests/rbac.yaml | envsubst | kubectl apply -f -
}


# wlhe@: is it needed?
function setup_fortio() {
  helm upgrade --install \
      -n fortio \
      fortio \
      samples/charts/fortio
}

# wlhe@: GOLDEN_IMAGE needs to be published in a public repo.
function build_fortio() {
  docker build -f samples/fortio/Dockerfile . -t ${IMAGE} --build-arg=BASE=${GOLDEN_IMAGE} 
  docker push ${IMAGE}
}

function deploy_app() {
  gcloud alpha run deploy ${CLOUDRUN_SERVICE} \
          --platform managed \
          --project ${PROJECT_ID} \
          --region ${REGION} \
          --execution-environment=gen2 \
          --allow-unauthenticated \
          --use-http2 \
          --port 15009 \
          --image ${IMAGE} \
          --vpc-connector projects/${PROJECT_ID}/locations/${REGION}/connectors/serverlesscon \
          --service-account=k8s-${WORKLOAD_NAMESPACE}@${PROJECT_ID}.iam.gserviceaccount.com \
          --set-env-vars="CLUSTER_NAME=${CLUSTER_NAME}" \
          --set-env-vars="CLUSTER_LOCATION=${CLUSTER_LOCATION}"
}

