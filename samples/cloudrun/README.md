# Using krun with CloudRun and ASM

## Installation 

1. Install ASM ( managed )

https://cloud.google.com/service-mesh/docs/scripted-install/gke-install


```shell
export PROJECT_ID=wlhe-cr
export CLUSTER_LOCATION=us-central1-c
export CLUSTER_NAME=asm-cr
export REGION=us-central1
curl https://storage.googleapis.com/csm-artifacts/asm/install_asm_1.10 > install_asm
chmod +x install_asm

# Managed CP:
./install_asm --mode install --output_dir ${CLUSTER_NAME} --enable_all --managed
```




2. Install CloudRun connector (once per project)

``` 
	gcloud services enable --project ${PROJECT_ID} vpcaccess.googleapis.com
	gcloud compute networks vpc-access connectors create serverlesscon \
    --project ${PROJECT_ID} \
    --region ${REGION} \
    --subnet default \
    --subnet-project ${PROJECT_ID} \
    --min-instances 2 \
    --max-instances 10 

```

3. Deploy an in-cluster application

```
gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}

kubectl create ns fortio
kubectl label namespace fortio istio-injection- istio.io/rev=asm-managed-rapid --overwrite

helm upgrade --install \
		-n fortio \
		fortio \
 		samples/charts/fortio

```

4. Create a google service account for the CloudRun app (once per project)


```shell
export NS=fortio # Namespace 

gcloud --project ${PROJECT_ID} iam service-accounts create k8s-${NS} \
      --display-name "Service account with access to ${NS} k8s namespace"

gcloud --project ${PROJECT_ID} projects add-iam-policy-binding \
            ${PROJECT_ID} \
            --member="serviceAccount:k8s-${NS}@${PROJECT_ID}.iam.gserviceaccount.com" \
            --role="roles/container.clusterViewer"


```

5. Bind the GSA to a KSA

```shell 

cat samples/fortio/rbac.yaml | envsubst | kubectl apply -f -

```

## Build a docker image containing the app and the sidecar

samples/fortio/Dockerfile contains an example Dockerfile, with comments. 

You can build the app with the normal docker command:

```shell

export IMAGE=gcr.io/${PROJECT_ID}/fortio-cr:latest
(cd samples/fortio && docker build . 	-t ${IMAGE})
docker push ${IMAGE}
```

## Deploy the image to CloudRun

```shell

```
