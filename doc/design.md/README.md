# krun

Run a command in a K8S and Istio environment, similar with the Pod environment created by
kubelet. The goal is to minimize the differences and pain associated with running Istio and K8S-first
application on VMs or Docker containers outside K8S.

This includes a CLI and small library that perform one or more of the following steps:

- use KUBECONFIG or platform credentials and APIs to find the K8S APIserver and construct a 'kubeconfig' (currently GCP only)
- 'mount' audience-based tokens in the expected locations, including token for api server and istio.
- 'mount' additional secrets and config maps.
- create the environment (files, etc) expected by Istio in K8S 
- start Istio. If the command was started as root, Istio will run as user 1337 and iptables will be initialized,
  otherwise Istio will run as the current user, in 'whitebox' mode (no iptables).
- start the app
- optionally start an cert-based in-process SSH server for debugging, providing equivalent of kubectl exec, copy
and port forward. 
- periodically refreshes the K8S token and other resources, similar with kubelet.
  

All krun configurations are based environment variables and env detection (including metadata server, config maps in 
the cluster) - the parameters on the command line are passed directly to the application.


## Running as non-root

KRun can also be used as regular user, howerver:

- iptables will not be set
- envoy (if found) will run with the current UID
- if envoy is not found, pilot-agent will still be started and generate proxyless gRPC config and certs
- all files will be created relative to current dir instead of root dir.
- Istio will use interception mode NONE - this enables 127.0.0.1:PORT bindings for mesh TCP services.

In this mode Istio can't capture traffic - it works in 'whitebox' mode, using HTTP_PROXY environment variable to 
capture HTTP and Sidecar API for forwarding local ports to services. 

It currently requires MeshConfig HttpProxyPort to be set - in 1.12 this will be automatically set for the workload,
no need for global config (PR#...)

Non-root mode is useful in Docker environments where iptables and/or running as root are not possible. For example
CI/CDs, current CloudRun VMs (minivm supports iptables), developer machine.  

## Authentication

Connection to K8S and Istiod will authenticate using:

- metadata server or GOOGLE_APPLICATION_CREDENTIALS for GKE
- an existing KUBECONFIG or $HOME/kube/config 

In the first case, the Google Service Account requires the appropriate permissions. 


# Configuration options and defaults

## GKE support

When running on a GCP VM, CloudRun instance or a VM with access to downloaded Google Service
Account credentials, the library can get the APIserver URL and certificate, and create a kube config file.

Credentials can be provided by a local metadata server or downloaded service account.
The SA must have the correct IAM permissions.

- `CLUSTER_NAME` - name of the GKE or Hub cluster
- `CLUSTER_LOCATION` - optional, if specified, zone or region of the GKE cluster. By default metadata server is used to find
  the region of the workload, and the cluster is looked for in the same region.
- `PROJECT_ID` - optional, the project of the GKE cluster. By default metadata server is used (same project as the workload)

For authentication metadata server or GOOGLE_APPLICATION_CREDENTIALS will be used.

Work in progress to support Hub and the Hub connector.

## K8S namespace mapping

The workload needs to know the namespace and KSA.

- `WORKLOAD_NAMESPACE` 
- `WORKLOAD_NAME` - it will default to the same value with WORKLOAD_NAMESPACE
- `WORKLOAD_SERVICE_ACCOUNT` - default is "default"

If the workload runs in CloudRun and namespace/name are not set, K_SERVICE can be used to infer the namespace and name,
will be parsed as [$VERSION--]$WORKLOAD_NAMESPACE[-$WORKLOAD_NAME]

## Local debugging

When running on a local docker or dev machine, since metadata server is not available:

- GOOGLE_APPLICATION_CREDENTIALS must be set and the content must be mounted/available
- CLUSTER_LOCATION and PROJECT_ID are required
- WORKLOAD_NAMESPACE or K_SERVICE are required

# Similar projects, other ideas

WIP to incorporate some ideas and UX, for consistency - and to replace equivalent 
functionality. I discovered the projects too late.

- https://github.com/kelseyhightower/konfig 
  - uses CloudRun API to get the Service manifest with  
    the original env
  - directly connect to APIServer using http client - just get.
  - can get ConfigMap and Secret from ANY cluster (if RBAC is in place)

UX: 
```shell
CLUSTER_ID=/projects/hightowerlabs/zones/us-central1-a/clusters/k0

gcloud run ...
  -set-env-vars FOO=\$SecretKeyRef:${CLUSTER_ID}/namespaces/default/secrets/env/keys/foo,CONFIG_FILE=\$SecretKeyRef:${CLUSTER_ID}/namespaces/default/secrets/env/keys/config.json?tempFile=true,ENVIRONMENT=\$ConfigMapKeyRef:${CLUSTER_ID}/namespaces/default/configmaps/env/keys/environment"
```

- https://github.com/GoogleCloudPlatform/berglas
  - replaces env variable using berglas://<bucket>/<secret>?<params>
  - destination=path to write to file
  - secrets only, using Cloud KMS or Secret Manager.

- https://ahmet.im/blog/cloud-run-deploy-api/ - useful for automating testing and possibly controlling scale automatically from the SNI gate.

- https://github.com/ahmetb/runsd 
  - DNS interception - redirects to localhost, http transparent proxy
  - includes a magic map of region codes
  - gets the JWT, use regular TLS
  - `http://<SERVICE_NAME>.<REGION>`
