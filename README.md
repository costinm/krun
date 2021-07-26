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
  

All CLI configuration uses environment variables and env detection,
the remaining of the command line is used to start the application.


## Running as non-root

KRun can also be used as regular user, howerver:

- iptables will not be set
- envoy (if found) will run with the current UID
- all files will be created relative to current dir instead of root dir.

In this mode Istio can't capture traffic - however it can work in 'whitebox' mode,
using HTTP_PROXY environment variable to capture HTTP and Sidecar API for forwarding
local ports to services. 

## Authentication


## GKE support 

When running on a GCP VM, CloudRun instance or a VM with access to downloaded Google Service
Account credentials, the library can get the APIserver URL and certificate, and create a kube config file.

Credentials can be provided by a local metadata server or downloaded service account.
The SA must have the correct IAM permissions. 

- `CLUSTER` - name of the GKE or Hub cluster
- `LOCATION` - if specified, will look for a GKE cluster in that location. Otherwise
 will use Hub.
- 

## Anthos Connect Gateway

This also works for GKE Connect Gateway, for private or non-GKE clusters registered in the hub.

