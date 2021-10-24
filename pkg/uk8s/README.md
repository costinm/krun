WIP, not used yet: using the rest API directly to fetch mesh-env and tokens.

Based on a subset of kelseyhightower/konfig.

All we really need is getting a config map with a GET request, and creating a token with a POST request - both using
JWTs from the default credentials source. We also don't need the full json - just few fields that are stable, so keeping
a dependency to the full generated structs of all k8s APIs is overkill.

This includes minimal code to parse kubeconfig, just enough for debugging.

The 'primary' auth mode is using google service account, using golang.org/x/oauth2/google:
- metadata server
- GOOGLE_APPLICATION_CREDENTIALS
- $HOME/.config/gcloud/application_default_credentials.json


# Cluster Discovery

The code is currently 'optimized' for GCP, but can be extended to any similar provider, if a REST 'discovery' API is
provided. We use the container API to list the clusters in the same project, and select a cluster in the same region
based on labels (falling back to other regions if the local one is not available).

TODO: document selection for config clusters
TODO: document hub discovery

#  Others

https://github.com/ericchiang/k8s
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
