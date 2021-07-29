# Using krun with CloudRun and ASM

## Installation 

### Project and cluster setup

1. Install ASM ( managed )

See [Install docs](https://cloud.google.com/service-mesh/docs/scripted-install/gke-install) for ASM, make sure
all requirements are met. For googlers make sure the project is allowed to use 'allow-unauthenticated'.


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



2. Install CloudRun connector

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

### Namespace setup 

The steps can be run by a user or service account with namespace permissions in K8S and CR deploy permission.

1. Create a google service account for the CloudRun app (once per namespace)


```shell
export NS=fortio # Namespace 

gcloud --project ${PROJECT_ID} iam service-accounts create k8s-${NS} \
      --display-name "Service account with access to ${NS} k8s namespace"

gcloud --project ${PROJECT_ID} projects add-iam-policy-binding \
            ${PROJECT_ID} \
            --member="serviceAccount:k8s-${NS}@${PROJECT_ID}.iam.gserviceaccount.com" \
            --role="roles/container.clusterViewer"


```

2. Bind the GSA to a KSA
   You can grant additional permissions if the CloudRun service is using the K8S ApiServer. To keep things simple, we
   associate with the 'default' KSA in the namespace.

```shell 
export PROJECT_ID=wlhe-cr
export NS=fortio 

cat manifests/rbac.yaml | envsubst | kubectl apply -f -

```

### Build a docker image containing the app and the sidecar

samples/fortio/Dockerfile contains an example Dockerfile, with comments.

You can build the app with the normal docker command:

```shell

export IMAGE=gcr.io/${PROJECT_ID}/fortio-cr:latest
(cd samples/fortio && docker build . 	-t ${IMAGE})
docker push ${IMAGE}
```



### Deploy the image to CloudRun

WARNING: WIP to eliminate this step - either on Thetis side or using a ConfigMap in cluster ( where other settings
can be defined ).
Also WORKLOAD_NAMESPACE, WORKLOAD_NAME can be derived from the cloudrun service - for example using a default naming scheme.


```shell

export MCP_ADDR="$(kubectl get mutatingwebhookconfiguration istiod-asm-managed -ojson | jq .webhooks[0].clientConfig.url -r | cut -d'/' -f3)"

```

Deploy the service:


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
         --set-env-vars="WORKLOAD_NAMESPACE=fortio" \
         --set-env-vars="WORKLOAD_NAME=fortio-cr" \
         --set-env-vars="LABEL_APP=fortio-cr"
         
```

### Testing

1. Deploy an in-cluster application

```
gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}

kubectl create ns fortio
kubectl label namespace fortio istio-injection- istio.io/rev=asm-managed-rapid --overwrite

helm upgrade --install \
		-n fortio \
		fortio \
 		samples/charts/fortio

```


2. Use the CloudRun service to connect to the in-cluster workload. Use the CR service URL with /fortio/ path to
access the UI of the app.

In the UI, use "http://fortio.fortio.svc:8080" and you should see the results for testing the connection to the 
in-cluster app.

In general, the CloudRun applications can use any K8S service name - including shorter version for same-namespace
services. So fortio, fortio.fortio, fortio.fortio.svc.cluster.local also work.


## Configuration options 

Configuration is based on environment variables and metadata server. 

We expect a GKE cluster in the same project with the CloudRun service. CLUSTER_NAME and CLUSTER_LOCATION allow 
finding the cluster. The init steps grant the GSA running the service (minimal) access to the cluster. 

WORKLOAD_NAMESPACE and WORKLOAD_NAME (TODO: use shorter names) map the CloudRun service to the equivalent of a k8s pod, 
in the given namespace. You can specify additional labels to be used by Istio for config generation - in Istio
configs associate with workloads using label selectors.

The MCP_ADDR is a temporary requirement - will be replaced with an automated mechanism.

Currently 'sandbox=minivm' is required for iptables. It is possible to run the same thing in gvisor, usign the 
istio agent http proxy.

'--use-http2' and '--port 15009' are required for using the 'hbone' port multiplexing. The app is still expected to
run on port 8080. It is also possible to not set the flags and use the normal CloudRun ingress - debugging will not
be possible. '--allow-unauthenticated' is also only needed if tunnel mode - where mTLS is expected for authentication
(WIP). 



# Debugging

By adding `--set-env-vars="SSH_AUTH=$(cat ~/.ssh/id_ecdsa.pub)"` you enable a built-in ssh server that will
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
