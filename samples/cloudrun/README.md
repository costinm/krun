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

WARNING: WIP to eliminate this step - either on Thetis side or using a ConfigMap in cluster ( where other settings 
can be defined )

```shell

export MCP_ADDR="$(kubectl get mutatingwebhookconfiguration istiod-asm-managed -ojson | jq .webhooks[0].clientConfig.url -r | cut -d'/' -f3)"

```




```shell
export CLOUDRUN_SERVICE=fortio-asm-cr
export CLOUDRUN_REGION=us-central1

gcloud alpha run deploy ${CLOUDRUN_SERVICE} \
          --platform managed 
          --project ${PROJECT_ID} \
          --region ${CLOUDRUN_REGION} \
          --sandbox=minivm \
          --allow-unauthenticated \
          --use-http2 \
          --port 15009 \
          --image ${IMAGE} \
          --vpc-connector projects/${PROJECT_ID}/locations/${CLOUDRUN_REGION}/connectors/serverlesscon
         --set-env-vars="CLUSTER_NAME=asm-cr" \
         --set-env-vars="CLUSTER_LOCATION=us-central1-c" \
         --set-env-vars="ISTIO_META_CLOUDRUN_ADDR=${MCP_ADDR}" \
         --set-env-vars="POD_NAMESPACE=fortio" \
         --set-env-vars="POD_NAME=fortio-cr" \
         --set-env-vars="LABEL_APP=fortio-cr"
         
```



# Test the image

The fortio example is accessible on the cloudrun URL as /fortio/ - in the UI enter 

"http://fortio.fortio.svc:8080" and you should see the results for testing the connection to the in-cluster app.

In general, the CloudRun applications can use any K8S service name - including shorter version for same-namespace 
services. So fortio, fortio.fortio, fortio.fortio.svc.cluster.local also work.


# Debugging

By adding `--set-env-vars="SSH_AUTH=$(shell cat ~/.ssh/id_ecdsa.pub)"` you enable a built-in ssh server that will
allow connections using your local ssh key. Make sure `ssh-keygen -t ecdsa` was run if the file is missing.

You can ssh into the service and forward ports - like envoy admin port - using:

```shell

# Compile the proxy command
go install ./cmd/hbonec

# Set with your own service URL
export SERVICE_URL=https://fortio-asm-cr-icq63pqnqq-uc.a.run.app:443

ssh -F /dev/null -o StrictHostKeyChecking=no -o "UserKnownHostsFile /dev/null" \
    -o ProxyCommand='hbone ${SERVICE_URL}/_hbone/22' root@proxy
```
