# uRest

This package includes a basic wrapper around http Client interface, to support common 'REST' API for GCP, K8S and 
other similar APIs without a full dependency on the (rather large) client libraries.

It is ONLY intended for bootstrap and 'light' usage - and optimized for code size instead of performance.

uRest uses the basic HTTP protocol with an opaque payload which can be further interpreted as json, protobuf, 
certificates or other formats. 

# Background

This is using the k8s rest API directly to fetch config maps, secrets and tokens.

Originally based on a subset of kelseyhightower/konfig - a subset of the supported objects is defined as a json struct.

For bootstrap, all we really need is getting a config map with a GET request, and creating a token with a POST request - both using
JWTs from the default credentials source. We also don't need the full json - just few fields that are stable, so keeping
a dependency to the full generated structs of all k8s APIs is overkill.

This includes minimal code to parse kubeconfig, just enough for debugging and running in environments without a metadata server.

The 'primary' auth mode is using google service account, using golang.org/x/oauth2/google:
- metadata server
- GOOGLE_APPLICATION_CREDENTIALS

## Authentication

All authentication starts with a 'source of trust' - usually a secret.

The source of trust is:
- Existing key/certificate, using the default location used in GKE.
- a KUBECONFIG or ~/.kube.config file. All clusters are auto-loaded.
- GCP GOOGLE_APPLICATION_CREDENTIALS . 
- $HOME/.config/gcloud/application_default_credentials.json - as used by golang.org/x/oauth2/google
- metadata server - second last to allow explicit override.
- in-cluster - it is last to allow pods running in a cluster to connect to other clusters and 
  override the default

This includes minimal code to parse kubeconfig, just enough for bootstrap when using
JWT tokens or GCP.

This root secrets are exchanged for other tokens and certificates:
- the mesh certificate - using Citadel or CAS protocols
- domain certs - using STS
- K8S scoped tokens
- GCP access tokens for the federated identity
- GCP access tokens for any GSA that allows it.
- GCP OIDC tokens


# Cluster Discovery

The code is currently 'optimized' for GCP, but can be extended to any similar provider, if a REST 'discovery' API is
provided. We use the container API to list the clusters in the same project, and select a cluster in the same region
based on labels (falling back to other regions if the local one is not available).

TODO: document selection for config clusters
TODO: document hub discovery

#  Others

kelseyhightower/konfig
- base for this package 
- specialized for a specific function

K8S-only: https://github.com/ericchiang/k8s
- archived
- using the protobuf option instead of JSON - would be good for high-perf clients, we only need few configs
- generated
- custom resources are still JSON
- in-cluster client 
- kubeconfig struct forked, similar code to load
- TODO: simple 'watch' interface, may be worth forking just this

Main difference is that instead of generating ALL k8s protos and using them, this
library is only manually including a specialized small subset and allow using raw json.

It does not support 'from metadata' (GKE) or 'exec' auth - but supports client cert, token, user/pass.

https://github.com/go-resty/resty
- deps free
- backoff/retry
- gopkg.in/resty.v1 is a dep of istio already

k8s.io/client-go/rest/client.go
- includes rate limitter golang.org/x/time/rate
- backoff


# TODO

[] Evaluate using same naming conventions and style with swagger-generated
[] Evaluate using same naming as gRPC, possibly same transports.
