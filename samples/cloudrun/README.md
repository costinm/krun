# Using krun with CloudRun and ASM

## Installation 

### Cluster setup

1. Install Managed ASM in the GKE config cluster
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



2. Allow read access to mesh config:

```shell 

kubectl apply -f manifests/mcp-rbac.yaml

```

### Connector setup

For each region where GKE and CloudRun will be used, [install CloudRun connector](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access)
Using the UI is usually easier - it does require a /28 range to be specified.
You can call the connector 'serverlesscon' - the name will be used
when deploying the CloudRun service.


### Namespace setup 

The steps can be run by a user or service account with namespace permissions in K8S and CR deploy permission.
In this example we will use namespace 'fortio', set as NS env variable.

1. Create a google service account for the CloudRun app (once per namespace)


```shell
export NS=fortio # Namespace 

kubectl create ns ${NS}

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
export NS=fortio 

cat manifests/rbac.yaml | envsubst | kubectl apply -f -

```

### Build a docker image containing the app and the sidecar

samples/fortio/Dockerfile contains an example Dockerfile - you can also use the pre-build image
`grc.io/wlhe-cr/fortio-cr:latest`

You can build the app with the normal docker command:

```shell

# Get the base image.
docker pull ghcr.io/costinm/krun/krun:latest

# Target image
export IMAGE=gcr.io/${PROJECT_ID}/fortio-cr:latest

(cd samples/fortio && docker build . -t ${IMAGE})

docker push ${IMAGE}
```



### Deploy the image to CloudRun

Deploy the service, with explicit configuration:

```shell
export CLOUDRUN_SERVICE=fortio-cloudrun
export REGION=us-central1

gcloud alpha run deploy ${CLOUDRUN_SERVICE} \
          --platform managed \
          --project ${PROJECT_ID} \
          --region ${REGION} \
          --sandbox=minivm \
          --serviceAccount:k8s-${NS}@${PROJECT_ID}.iam.gserviceaccount.com \
          --allow-unauthenticated \
          --use-http2 \
          --port 15009 \
          --image ${IMAGE} \
          --vpc-connector projects/${PROJECT_ID}/locations/${CLOUDRUN_REGION}/connectors/serverlesscon \
         --set-env-vars="CLUSTER_NAME=${CLUSTER_NAME}" \
         --set-env-vars="CLUSTER_LOCATION=${CLUSTER_LOCATION}" 
         
```

CLUSTER_NAME and CLUSTER_LOCATION will be optional - krun will pick a config cluster in the same region based on a TBD 
label, and fallback to other config cluster if the local cluster is unavailable.

### Testing

1. Deploy an in-cluster application

```
gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}

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

- CLUSTER_NAME - name of the config cluster, required. 
- CLUSTER_LOCATION - location of the GKE config cluster. Optional if 
  the cluster is regional and in same region with the CloudRun service.
- PROJECT_ID - project of the config cluster - optional, defaults to same
project.




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
