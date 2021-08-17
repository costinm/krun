# KRun testing

The core functionality of KRun is to integrate with K8S and GKE API and create a 
pod-like environment. As such, testing will require a GCP project with a GKE
cluster.

## Testing plan

- integration with GKE and getting a k8s config using a GSA
- integration using KUBECONFIG
- token rotation
- mapping secrets and configmaps
- locating istiod config
- connection to MCP
- generated proxyless gRPC config

The test will first build krun and run each test scenario locally.
A docker image including a test app will be deployed as a set of CloudRun
services, and tested using direct http and tunneled mTLS.

## Test env

Currently using wlhe-cr project, with istio cluster. Eventually should
use Prow.



