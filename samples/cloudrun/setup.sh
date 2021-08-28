#!/bin/bash

export PROJECT_ID=${PROJECT_ID:-wlhe-cr}
export CLUSTER_LOCATION=${CLUSTER_LOCATION:-us-central1-c}
export CLUSTER_NAME=${CLUSTER_NAME:-asm-cr}
export REGION=${REGION:-us-central1}

export NS=${NS:-fortio}

# Example command to create a regular cluster.
function create_cluster() {

  gcloud beta container --project "${PROJECT_ID}" clusters create \
    "${CLUSTER_NAME}" --zone "${CLUSTER_LOCATION}" \
    --no-enable-basic-auth \
    --cluster-version "1.20.8-gke.700" \
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


function setup_namespace() {

  gcloud --project ${PROJECT_ID} iam service-accounts create k8s-${NS} \
        --display-name "Service account with access to ${NS} k8s namespace"

  gcloud --project ${PROJECT_ID} projects add-iam-policy-binding \
              ${PROJECT_ID} \
              --member="serviceAccount:k8s-${NS}@${PROJECT_ID}.iam.gserviceaccount.com" \
              --role="roles/container.clusterViewer"

  cat manifests/rbac.yaml | envsubst | kubectl apply -f -
}

function setup_fortio() {
  gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}

  kubectl create ns fortio
  kubectl label namespace fortio istio-injection- istio.io/rev=asm-managed-rapid --overwrite

  helm upgrade --install \
      -n fortio \
      fortio \
      samples/charts/fortio
}


function deploy_app() {
  export CLOUDRUN_SERVICE=fortio-asm-cr
  export REGION=us-central1

  gcloud alpha run deploy ${CLOUDRUN_SERVICE} \
          --platform managed \
          --project ${PROJECT_ID} \
          --region ${REGION} \
          --sandbox=minivm \
          --allow-unauthenticated \
          --use-http2 \
          --port 15009 \
          --image ${IMAGE} \
          --vpc-connector projects/${PROJECT_ID}/locations/${CLOUDRUN_REGION}/connectors/serverlesscon \
          --set-env-vars="CLUSTER_NAME=${CLUSTER_NAME}" \
         --set-env-vars="CLUSTER_LOCATION=${CLUSTER_LOCATION}"
}
