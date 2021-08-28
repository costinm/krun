# Running a CloudRun or docker image in a mesh environment

This repository implements a small launcher that prepares a mesh environment and starts the user application.

In K8S, the mesh implementation relies on a mutating webhook that patches the Pod, injecting for required environment. 
Docker and CloudRun images do not have an injector - this application is playing the same role, using the K8S and
GCP APIs to setup iptables and the sidecar process or the proxyless bootstrap.

This supports running an app:
- in an Istio-like environment with a Sidecar and iptables interception
- if iptables is not available (gVisor, regular docker, dev machine), configure 'whitebox' mode ( HTTP_PROXY and localhost port 
  forwarding for TCP)
- in proxyless gRPC mode for applications that natively support XDS and mesh, without iptables or sidecar.

The launcher is responsible for:
- discovering a GKE/K8S cluster based on environment (metadata server, env variables), and getting credentials and config
- discovering the XDS address and config (root certificates, metadata)
- setting up iptables ( equivalent to the init container in K8S ), using pilot-agent
- launching pilot-agent and envoy
- configuring pilot-agent to intercept DNS
- launching the application - after the setup is ready
- creating any necessary tunnels to allow use of mTLS, based on the HBONE (tunneling over HTTP/2) proposal in Istio.

The repository also includes a specialised SNI-routing gateway that allows any mesh node using mTLS to route back 
to CloudRun with the proper authentication and tunneling for mTLS.

The user application should be able to communicate with other mesh workloads - in Pods, VMs or other CloudRun 
services using mTLS and the mesh launcher.

The code is based on the Istio VM startup script and injection template and will need to be kept in sync with future
changes in the mesh startup.

# Setup instructions

Common environment variables used in this document:

```shell

export PROJECT_ID=<PROJECT_ID>
export CLUSTER_LOCATION=us-central1-c
export CLUSTER_NAME=asm-cr
# CloudRun region 
export REGION=us-central1

export WORKLOAD_NAMESPACE=fortio # Namespace where the Cloud Run service will 'attach'
export WORKLOAD_NAME=cloudrun

# Name of the service account running the Cloud Run service. It is recommended to use a dedicated SA for each K8S namespace
# and keep permissions as small as possible. 
!!! wlhe@: what is GSA? Google Service account?
# By default the namespace is extracted from the GSA name - if using a different SA or naming, WORKLOAD_NAMESPACE env
# is required when deploying the docker image. 
# (This may change as we polish the UX)
export WORKLOAD_SERVICE_ACCOUNT=k8s-${WORKLOAD_NAMESPACE}@${PROJECT_ID}.iam.gserviceaccount.com

# Name for the Cloud Run service - will use the same as the workload.
# Note that the service must be unique for the region, if you want the same name in multiple namespace you must 
# use explicit config for WORKLOAD_NAME when deploying a unique Cloud Run service name
export CLOUDRUN_SERVICE=${WORKLOAD_NAME}


````


## Installation 

Requirements:
- For each region, you need a Serverless connector, using the same network as the GKE cluster(s) and VMs. Cloud Run will
   use it to communicate with the Pods/VMs over VPC.
- 'gen2' VM required for iptables. 'gen1' works in 'whitebox mode', using HTTP_PROXY. 
- The project should be allowed by policy to use 'allow-unauthenticated'. WIP to eliminate this limitation.

You need to have gcloud and kubectl, and admin permissions for the project and cluster. 

After installation, new services can be configured for namespaces using only namespace-level permissions in K8S.


### Cluster setup (once per cluster)

!!!
0. create a new cluster
source samples/cloudrun/setup.sh && create_cluster

1st run:
Error: (gcloud.beta.container.clusters.create) ResponseError: code=400, message=Master version must be one of "RAPID" channel supported versions [1.20.8-gke.2100, 1.20.9-gke.2100, 1.21.3-gke.901, 1.21.3-gke.1100, 1.21.3-gke.2000].

Solution: update the script

2nd run:
ERROR: (gcloud.beta.container.clusters.create) ResponseError: code=400, message=IP aliases cannot be used with a legacy network.

Create a new project.
Enable GCE API to have a new default subnet.
gcloud services enable compute.googleapis.com --project wlhe-asm

Enable GKE API. 
gcloud services enable container.googleapis.com --project wlhe-asm
!!!

1. If you don't already have a cluster with managed ASM, follow [Install docs](https://cloud.google.com/service-mesh/docs/scripted-install/gke-install) 

Short version:

source samples/cloudrun/setup.sh && setup_asm

```shell
curl https://storage.googleapis.com/csm-artifacts/asm/install_asm_1.10 > install_asm
chmod +x install_asm

./install_asm --mode install --output_dir ${CLUSTER_NAME} --enable_all --managed
```

!!! 1st run:
!!! wlhe-macbookpro:k8s_install wlhe$ ./install_asm --mode install --output_dir ${CLUSTER_NAME} --enable_all --managed
WARNING: bash 3.2.57(1)-release does not support several modern safety features.
This script was written with the latest POSIX standard in mind, and was only
tested with modern shell standards. This script may not perform correctly in
this environment.
install_asm: Setting up necessary files...
install_asm: Using asm_kubeconfig as the kubeconfig...
install_asm: Fetching/writing GCP credentials to kubeconfig file...
install_asm: [WARNING]: Failed, retrying...(1 of 2)
install_asm: [WARNING]: Failed, retrying...(2 of 2)

Solution: create a new cluster, see step 0.


2nd run:
wlhe-macbookpro:asm wlhe$ source samples/cloudrun/setup.sh && setup_asm
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100 88918  100 88918    0     0   304k      0 --:--:-- --:--:-- --:--:--  304k
WARNING: bash 3.2.57(1)-release does not support several modern safety features.
This script was written with the latest POSIX standard in mind, and was only
tested with modern shell standards. This script may not perform correctly in
this environment.

install_asm: Setting up necessary files...
install_asm: Using asm_kubeconfig as the kubeconfig...
install_asm: Fetching/writing GCP credentials to kubeconfig file...
install_asm: Verifying connectivity (10s)...
install_asm: kubeconfig set to asm_kubeconfig
install_asm: context set to gke_wlhe-asm_us-central1-c_asm-cr
install_asm: Checking installation tool dependencies...
install_asm: Fetching/writing GCP credentials to kubeconfig file...
install_asm: Verifying connectivity (10s)...
install_asm: kubeconfig set to asm_kubeconfig
install_asm: context set to gke_wlhe-asm_us-central1-c_asm-cr
install_asm: Getting account information...
install_asm: Downloading kpt..
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   638  100   638    0     0   2091      0 --:--:-- --:--:-- --:--:--  2091
100 11.9M  100 11.9M    0     0  13.9M      0 --:--:-- --:--:-- --:--:-- 13.9M
install_asm: Downloading ASM..
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100 40.3M  100 40.3M    0     0  31.9M      0  0:00:01  0:00:01 --:--:-- 31.9M
install_asm: Downloading ASM kpt package...
Segmentation fault: 11

Solution: switch to linux rather than using mac
!!!


2. Allow read access to mesh config. This is needed to simplify the configuration - it is also possible to 
   explicitly pass extra env variables to the CloudRun services instead of using this config, but it is simpler to just
   directly parse the in-cluster config. This step is temporary, WIP to remove it.

```shell 

kubectl apply -f manifests/istio-system-discovery-rbac.yaml

```

### Connector setup (once per project)

For each region where GKE and CloudRun will be used, [install CloudRun connector](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access)
Using the UI is usually easier - it does require a /28 range to be specified.
You can call the connector 'serverlesscon' - the name will be used when deploying the CloudRun service. 

!!!
Example: gcloud compute networks subnets create vpc-us-central1 --region us-central1 --network default --range 10.1.1.0/28 -
-project $PROJECT_ID
!!!

If you already have a connector, you can continue to use it, and adjust the '--vpc-connector' parameter on the 
deploy command.

The connector MUST be on the same network with the GKE cluster.

!!!
### Setup Google service account
source samples/cloudrun/setup.sh && setup_namespace

1. Create a google service account for the CloudRun app (recommended - one per namespace, to reduce permission  scope).

2. Grant '--role="roles/container.clusterViewer"' to the service account.

```shell
gcloud --project ${PROJECT_ID} iam service-accounts create k8s-${WORKLOAD_NAMESPACE} \
      --display-name "Service account with access to ${WORKLOAD_NAMESPACE} k8s namespace"

gcloud --project ${PROJECT_ID} projects add-iam-policy-binding \
            ${PROJECT_ID} \
            --member="serviceAccount:${WORKLOAD_SERVICE_ACCOUNT}" \
            --role="roles/container.clusterViewer"
```

!!!

### Namespace setup 

Each CloudRun service will be mapped to a K8S namespace. The service account used by CloudRun must be granted access
to the GKE APIserver with minimal permissions, and must be allowed to get K8S tokens.

This steps can be run by a user or service account with namespace permissions in K8S - does not require k8s cluster admin.

3. Grant RBAC permissions to the google service account, allowing it to access in-namespace config map and use 
   TokenReview for the default KSA. (this step is also temporary, WIP to make it optional). This is used to get the MeshCA 
   certificate and communicate with the managed control plane - Istio injector is mounting the equivalent tokens. 

!!!
source samples/cloudrun/setup.sh && setup_namespace
!!!

```shell

gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}

kubectl create ns ${WORKLOAD_NAMESPACE}

# Uses WORKLOAD_NAMESPACE and WORKLOAD_SERVICE_ACCOUNT to grant permissions to the 'default' KSA in the namespace.
cat manifests/rbac.yaml | envsubst | kubectl apply -f -

```

### Build a docker image containing the app and the sidecar

samples/fortio/Dockerfile contains an example Dockerfile - you can also use the pre-build image
`gcr.io/wlhe-cr/fortio-cr:latest`

You can build the app with the normal docker command:

!!!
source samples/cloudrun/setup.sh && build_fortio
!!!

```shell

# Get the base image. You can also create a 'golden' base, starting with ASM proxy image and adding the 
# startup helper (krun) and other files or configs you need. 
# The application will be added to the base.
export GOLDEN_IMAGE=gcr.io/wlhe-cr/krun:main

# Target image 
export IMAGE=gcr.io/${PROJECT_ID}/fortio-cr:main

(cd samples/fortio && docker build . -t ${IMAGE} --build-arg=BASE=${GOLDEN_IMAGE} )

docker push ${IMAGE}
```



### Deploy the image to CloudRun

Deploy the service, with explicit configuration:

!!!
source samples/cloudrun/setup.sh && deploy_app
!!!


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

!!!
1st run:
Container failed to start at deployment:
2021-08-28T05:51:43.570964Z 2021-08-28T05:51:43.570376Z	info	Proxy role	ips=[169.254.8.1 169.254.8.130 169.254.1.2 fddf:3978:feb1:d745::c001 fe80::4000:a9ff:fefe:102] type=sidecar id=.cloudrun domain=cloudrun.svc.cluster.local A 
2021-08-28T05:51:43.571171Z 2021-08-28T05:51:43.570559Z	info	Apply proxy config from env {"discoveryAddress": ""} A 
2021-08-28T05:51:43.573291Z Error: failed to get proxy config: 1 error occurred: A 
2021-08-28T05:51:43.573301Z 2021-08-28T05:51:43.571744Z	error	failed to get proxy config: 1 error occurred: A 
2021-08-28T05:51:43.573304Z 	* discovery address must be set to the proxy discovery service A 
A 2021-08-28T05:38:08.817143Z 2021/08/28 05:38:08 Wait err  exit status 255 
A 2021-08-28T05:38:08.817423Z 2021/08/28 05:38:08 Failed to wait  /usr/bin/fortio server -http-port=8080 signal: killed 
  undefined


2nd run:
Use the pre-built image, but still fails to deploy with the same error.
!!!

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

When running in CloudRun, default automatic configuration is based on environment variables, metadata server and calls 
to GKE APIs. For debugging (as a regular process), when running with a regular docker or to override the defaults, the 
settings must be explicit.

- WORKLOAD_NAMESPACE - default value extracted from the service account running the CloudRun service
- WORKLOAD_NAME - default value is the CloudRun service name. Also used as 'canonical service'.
- PROJECT_ID - default is same project as the CloudRun service.
- CLUSTER_LOCATION - default is same region as the CloudRun service. If CLUSTER_NAME is not specified, a cluster with
  ASM in the region or zone will be picked.
- CLUSTER_NAME - if not set, clusters in same region or a zone in the region will be picked. Cluster names starting with
  'istio' are currently picked first. (WIP to define a labeling or other API for cluster selection)

Also for local development:
- GOOGLE_APPLICATION_CREDENTIALS must be set to a file that is mounted, containing GSA credentials. 
- Alternatively, a KUBECONFIG file must be set and configured for the intended cluster.


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
