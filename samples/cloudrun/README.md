# Using krun with CloudRun and ASM

## Installation 

Requirements:
- The project should be allowed by policy to use 'allow-unauthenticated'. WIP to eliminate this limitation.
- For each region, you need a Serverless connector, using the same network as the GKE cluster(s) the CloudRun service will
communicate with.
- Currently, 'sandbox=minivm' is required for iptables. 

You need to have gcloud and kubectl, and credentials for the cluster. Few environment variables will be used across
this doc:

```shell
export PROJECT_ID=wlhe-cr
export CLUSTER_LOCATION=us-central1-c
export CLUSTER_NAME=asm-cr
# CloudRun region 
export REGION=us-central1
export NS=fortio # Namespace where the CloudRun service will 'attach'

gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}

```

### Cluster setup

1. If you don't already have a cluster with managed ASM, follow [Install docs](https://cloud.google.com/service-mesh/docs/scripted-install/gke-install) 

Short version:

```shell
curl https://storage.googleapis.com/csm-artifacts/asm/install_asm_1.10 > install_asm
chmod +x install_asm

./install_asm --mode install --output_dir ${CLUSTER_NAME} --enable_all --managed
```


2. Allow read access to mesh config. This is needed to simplify the configuration - it is also possible to 
   explicitly pass extra env variables to the CloudRun services instead of using this config, will be documented in 
   separate doc for advanced users.

```shell 

kubectl apply -f manifests/mcp-rbac.yaml

```

### Connector setup

For each region where GKE and CloudRun will be used, [install CloudRun connector](https://cloud.google.com/vpc/docs/configure-serverless-vpc-access)
Using the UI is usually easier - it does require a /28 range to be specified.
You can call the connector 'serverlesscon' - the name will be used when deploying the CloudRun service. 

If you already have a connector, you can continue to use it, and adjust the '--vpc-connector' parameter on the 
deploy command.

The connector MUST be on the same network with the GKE cluster.


### Namespace setup 

Each CloudRun service will be mapped to a K8S namespace. The service account used by CloudRun must be granted access
to the GKE APIserver with minimal permissions, and must be allowed to get tokens.

This steps can be run by a user or service account with namespace permissions in K8S - does not require cluster admin.

In this example we will use namespace 'fortio', set as NS env variable.

1. Create a google service account for the CloudRun app (once per namespace). If you already have a GSA you use for 
your CloudRun service - only add '--role="roles/container.clusterViewer"' binding to the existing service account.


```shell

kubectl create ns ${NS}

gcloud --project ${PROJECT_ID} iam service-accounts create k8s-${NS} \
      --display-name "Service account with access to ${NS} k8s namespace"

gcloud --project ${PROJECT_ID} projects add-iam-policy-binding \
            ${PROJECT_ID} \
            --member="serviceAccount:k8s-${NS}@${PROJECT_ID}.iam.gserviceaccount.com" \
            --role="roles/container.clusterViewer"

```

2. Bind the GSA to a KSA. This will allow CloudRun service to get the required K8S resources to integrate with ASM.
   You can grant additional permissions if the CloudRun service is using the K8S ApiServer, if the application is also 
   integrating/using APIserver. 
   
   To keep things simple, we will associate with the 'default' KSA in the namespace, advanced users can customize the 
   config to use a different KSA.

```shell
export NS=fortio 
# Or use an existing GSA 
export GSA=k8s-${NS}@${PROJECT_ID}.iam.gserviceaccount.com

cat manifests/rbac.yaml | envsubst | kubectl apply -f -

```

### Build a docker image containing the app and the sidecar

samples/fortio/Dockerfile contains an example Dockerfile - you can also use the pre-build image
`grc.io/wlhe-cr/fortio-cr:latest`

You can build the app with the normal docker command:

```shell

# Get the base image. You can also create a 'golden' base, starting with ASM proxy image and adding the 
# startup helper (krun) and other files or configs you need. 
# The application will be added to the base.
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
          --service-account=k8s-${NS}@${PROJECT_ID}.iam.gserviceaccount.com \
          --allow-unauthenticated \
          --use-http2 \
          --port 15009 \
          --image ${IMAGE} \
          --vpc-connector projects/${PROJECT_ID}/locations/${REGION}/connectors/serverlesscon \
         --set-env-vars="CLUSTER_NAME=${CLUSTER_NAME}" \
         --set-env-vars="CLUSTER_LOCATION=${CLUSTER_LOCATION}" 
         
```

CLUSTER_NAME and CLUSTER_LOCATION will be optional - krun will pick a config cluster in the same region based on a TBD 
label, and fallback to other config cluster if the local cluster is unavailable.

### Testing

1. Deploy an in-cluster application. The CloudRun service will connect to it:

```shell
gcloud container clusters get-credentials ${CLUSTER_NAME} --zone ${CLUSTER_LOCATION} --project ${PROJECT_ID}

kubectl label namespace fortio istio-injection- istio.io/rev=asm-managed --overwrite
kubectl apply -f https://raw.githubusercontent.com/costinm/krun/main/samples/fortio/in-cluster.yaml

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

By adding `--set-env-vars="SSH_AUTH=$(cat ~/.ssh/id_ecdsa.pub)"` you enable a built-in ssh server that will
allow connections using your local ssh key. Make sure `ssh-keygen -t ecdsa` was run if the file is missing.

You can ssh into the service and forward ports - like envoy admin port - using:

```shell

# Compile the proxy command
go install ./cmd/hbone

# Set with your own service URL
export SERVICE_URL=https://fortio-asm-cr-icq63pqnqq-uc.a.run.app:443

ssh -F /dev/null -o StrictHostKeyChecking=no -o "UserKnownHostsFile /dev/null" \
    -o ProxyCommand='hbone ${SERVICE_URL}/_hbone/22' root@proxy
```
