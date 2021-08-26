# Running a CloudRun or docker image in a mesh environment

This repository implements a small launcher that prepares a mesh environemnt and launches a user application.

In K8S, Istio relies on a mutating webhook and injection for environment setup. Docker and CloudRun images do not 
have an injector - this application is playing the same role, using the K8S and GCP APIs to configure the application
and start the sidecar.

This supports:
- Istio-like environment with a Sidecar and iptables interception
- If iptables are not available (gVisor, regular docker), configure 'whitebox' mode ( HTTP_PROXY and localhost port forwarding for TCP)
- Based on setting, proxyless gRPC mode for applications that natively support XDS and mesh.

The app is responsible for:
- discovering a GKE/K8S cluster based on environment (metadata server, env variables)
- discovering the XDS address and config (root certificates, metadata)
- setting up iptables ( equivalent to the init container in K8S )
- launching pilot-agent and envoy
- launching the application
- creating any necessary tunnels to ensure mTLS is possible, based on the HBONE (tunneling over HTTP/2) proposal in Istio.

The repository also includes a specialised SNI gateway that allows any mesh node using mTLS to route back to CloudRun
with the proper authentication and tunneling for mTLS.

The application will communicate with other mesh workloads - in Pods, VMs or other CloudRun services - using mTLS and
behaving the same as any other Pod.


# Setup instructions

Common environment variables used in this document:

```shell

export PROJECT_ID=wlhe-cr
export CLUSTER_LOCATION=us-central1-c
export CLUSTER_NAME=asm-cr
# CloudRun region 
export REGION=us-central1

export WORKLOAD_NAMESPACE=fortio # Namespace where the CloudRun service will 'attach'
export WORKLOAD_NAME=cloudrun

# Derived - name is important, if using an existing GSA you must set WORKLOAD_NAMESPACE when deploying. 
# By default the namespace is extracted from the GSA name. 
# This may change as we polish the UX
export WORKLOAD_SERVICE_ACCOUNT=k8s-${WORKLOAD_NAMESPACE}@${PROJECT_ID}.iam.gserviceaccount.com

# Name for the cloudrun service - will use the same as the workload.
# Note that the service must be unique for region, if you want the same name in multiple namespace you must 
# use explicit config for WORKLOAD_NAME when deploying and unique cloudrun service name
export CLOUDRUN_SERVICE=${WORKLOAD_NAME}

````


## Installation 

Requirements:
- The project should be allowed by policy to use 'allow-unauthenticated'. WIP to eliminate this limitation.
- For each region, you need a Serverless connector, using the same network as the GKE cluster(s) the CloudRun service will
communicate with.
- 'gen2' VM required for iptables. 'gen1' works in 'whitebox mode', using HTTP_PROXY. 

You need to have gcloud and kubectl, and credentials for the cluster. 


### Cluster setup (once per cluster)

1. If you don't already have a cluster with managed ASM, follow [Install docs](https://cloud.google.com/service-mesh/docs/scripted-install/gke-install) 

Short version:

```shell
curl https://storage.googleapis.com/csm-artifacts/asm/install_asm_1.10 > install_asm
chmod +x install_asm

./install_asm --mode install --output_dir ${CLUSTER_NAME} --enable_all --managed
```

2. Allow read access to mesh config. This is needed to simplify the configuration - it is also possible to 
   explicitly pass extra env variables to the CloudRun services instead of using this config, but it is simpler to just
   directly parse the in-cluster config:

```shell 

kubectl apply -f manifests/istio-system-discovery-rbac.yaml

```

### Connector setup (once per project)

For each region where GKE and CloudRun will be used, [install CloudRun connector](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access)
Using the UI is usually easier - it does require a /28 range to be specified.
You can call the connector 'serverlesscon' - the name will be used when deploying the CloudRun service. 

If you already have a connector, you can continue to use it, and adjust the '--vpc-connector' parameter on the 
deploy command.

The connector MUST be on the same network with the GKE cluster.


### Namespace setup 

Each CloudRun service will be mapped to a K8S namespace. The service account used by CloudRun must be granted access
to the GKE APIserver with minimal permissions, and must be allowed to get K8S tokens.

This steps can be run by a user or service account with namespace permissions in K8S - does not require k8s cluster admin.


1. Create a google service account for the CloudRun app (once per namespace). If you already have a GSA you use for 
your CloudRun service - only add '--role="roles/container.clusterViewer"' binding to the existing service account.

2. Bind the GSA to a KSA. This will allow CloudRun service to get the required K8S resources to integrate with ASM.
   You can grant additional permissions if the CloudRun service is using the K8S ApiServer, if the application is also
   integrating/using APIserver.

   To keep things simple, we will associate with the 'default' KSA in the namespace, advanced users can customize the
   config to use a different KSA.


```shell

gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}


kubectl create ns ${WORKLOAD_NAMESPACE}

gcloud --project ${PROJECT_ID} iam service-accounts create k8s-${WORKLOAD_NAMESPACE} \
      --display-name "Service account with access to ${WORKLOAD_NAMESPACE} k8s namespace"

gcloud --project ${PROJECT_ID} projects add-iam-policy-binding \
            ${PROJECT_ID} \
            --member="serviceAccount:${WORKLOAD_SERVICE_ACCOUNT}" \
            --role="roles/container.clusterViewer"

# Uses WORKLOAD_NAMESPACE and WORKLOAD_SERVICE_ACCOUNT to grant permissions to the 'default' KSA in the namespace.
cat manifests/rbac.yaml | envsubst | kubectl apply -f -

```

### Build a docker image containing the app and the sidecar

samples/fortio/Dockerfile contains an example Dockerfile - you can also use the pre-build image
`grc.io/wlhe-cr/fortio-cr:main`

You can build the app with the normal docker command:

```shell

# Get the base image. You can also create a 'golden' base, starting with ASM proxy image and adding the 
# startup helper (krun) and other files or configs you need. 
# The application will be added to the base.
export GOLDEN_IMAGE=gcr.io/wlhe-cr/krun:main

docker pull ${GOLDEN_IMAGE}

# Target image 
export IMAGE=gcr.io/${PROJECT_ID}/fortio-cr:main

(cd samples/fortio && docker build . -t ${IMAGE} --build-arg=BASE=${GOLDEN_IMAGE} )

docker push ${IMAGE}

```



### Deploy the image to CloudRun

Deploy the service, with explicit configuration:


```shell

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
         
```

For versions of 'gcloud' older than 353.0, replace `--execution-environment=gen2` with `--sandbox=minivm`

CLUSTER_NAME and CLUSTER_LOCATION will be optional - krun will pick a config cluster in the same region  that is setup
with MCP, and fallback to other config cluster if the local cluster is unavailable. Cluster names starting with 'istio' 
will be used first in a region. (Will likely change to use a dedicated label on the project - WIP)

- `gcloud run deploy SERVICE --platform=managed --project --region` is common required parameters
- `--execution-environment=gen2` is currently required to have iptables enabled. Without it the 'whitebox' mode will
   be used (still WIP)
-  `--service-account` is recommended for 'minimal priviledge'. The service account will act as a K8S SA, and have its
   RBAC permissions
-   `--allow-unauthenticated` is only needed temporarily if you want to ssh into the instance for debug. WIP to fix this.
-  `--use-http2`  and `--port 15009` are required 

### Testing

1. Deploy an in-cluster application. The CloudRun service will connect to it:

```shell
gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}

kubectl label namespace fortio istio-injection- istio.io/rev=asm-managed --overwrite
kubectl apply -f https://raw.githubusercontent.com/costinm/cloud-run-mesh/main/samples/fortio/in-cluster.yaml

```


2. Use the CloudRun service to connect to the in-cluster workload. Use the CR service URL with /fortio/ path to
access the UI of the app.

In the UI, use "http://fortio.fortio.svc:8080" and you should see the results for testing the connection to the 
in-cluster app.

In general, the CloudRun applications can use any K8S service name - including shorter version for same-namespace
services. So fortio, fortio.fortio, fortio.fortio.svc.cluster.local also work.

In this example the in-cluster application is using ASM - it is also possible to access regular K8S applications
without a sidecar. 

## Configuration options 

Default automatic configuration is based on environment variables, metadata server and calls to GKE APIs. 

We require 2 environment variables - WIP to automatically locate the cluster:
- CLUSTER_NAME - name of the config cluster, required. 
- CLUSTER_LOCATION - location of the GKE config cluster. Optional if 
  the cluster is regional and in same region with the CloudRun service.
  
  
WORKLOAD_NAMESPACE and WORKLOAD_NAME map the CloudRun service to the equivalent of a k8s pod, 
in the given namespace. If not set, the CloudRun service name is used, the first part of the name
will be used as namespace, using the '-' as delimiter.

'--use-http2' and '--port 15009' are required for using the 'hbone' port multiplexing. The app is still expected to
run on port 8080. It is also possible to not set the flags and use the normal CloudRun ingress - debugging will not
be possible. '--allow-unauthenticated' is also only needed if tunnel mode - where mTLS is expected for authentication
(WIP). 



# Debugging

Since CloudRun and docker doesn't support kubectl exec or port-forward, we include a minimal sshd server that is 
enabled using a K8S Secret or environment variables. See samples/ssh for setup example. 

You can ssh into the service and forward ports using a regular ssh client and a ProxyCommand that implements 
the tunneling over HTTP/2:

```shell

# Compile the proxy command
go install ./cmd/hbone

# Set with your own service URL
export SERVICE_URL=https://fortio-asm-cr-icq63pqnqq-uc.a.run.app:443

ssh -F /dev/null -o StrictHostKeyChecking=no -o "UserKnownHostsFile /dev/null" \
    -o ProxyCommand='hbone ${SERVICE_URL}/_hbone/22' root@proxy
```
