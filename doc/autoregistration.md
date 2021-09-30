# Resource-based auto-registration

For KNative, users create a serving.knative.dev/v1/Service object, including the service name and template for the
docker image.

KNative creates the Service, Pods and mesh objects as needed.

CloudRun also support a mode where gcloud takes a yaml file, and applies the config. The protocol/API are actually the
same, except authn/z.

For Istio integration, one approach to auto-registration is to use a callback - for example when the pod starts, similar
with VM auto-registration. The registration service accepts the app credentials and auto-creates a set of config objects
- WorkloadInstance, etc. The problem with this approach is that we need to either give the app credentials too much
power, or we limit this to the bare config, i.e. the endpoint IP.

Another approach is to modify the CloudRun runtime to generate callbacks or pubsub events when a service is changed, and
use that to perform any config. The callbacks or topic will authenticate CloudRun SA, so we can have more trust and so
support more config options. The problem is that it'll take time and only works for CloudRun, can't be generalized for
arbitrary or local containers.

The third option is to keep using a set of yaml files - user deploys using gcloud (params or yaml), and applies a
template, with possible customizations. This provides maximum transparency and power - but the UX is a bit complicated,
a lot of boilerplate.

## Resource-based approach

One problem with 'gcloud run deploy' and the yaml equivalent is that while the protocol/format is similar with k8s, the
result is not reflected in the k8s cluster (or discovery system in general).

An alternative is to simply create a K8S resource - mirroring knative.Service, or even using knative.Service (if it has
an option to specify the runtime class - TBD). In the first case, it can be an extension to WorkloadGroup or a separate
Cloudrun-specific CRD.

A controller - like mesh gateway or Istiod or a CR-owned managed controller - will reconcile, just like in-cluster, and
automatically call 'gcloud run deploy' via API.

### Benefits

- we will have an in-cluster object that can be used for discovery, debug, etc - it can have Status, etc
- very easy to switch to/from in-cluster KNative.
- can be generalized - a dev docker instance or even VM only need to associate with the config using the service
  account, and will get all configs.
- deploying mesh CR services no longer requires 'gcloud' (which is pretty heavy). A possible issue or benefit is the
  swich to RBAC for controlling who can deploy - the controller will hold the IAM permissions for CloudRun. This may
  allow better isolation, i.e. a namespace owner may deploy services associated with the namespace (while someone with
  project-wide deploy permission has access to all namespaces)

# API design

## Requirements for registration

-

# Non-K8S mode

Cloudrun - and Istio - can also work without depenency on K8S. For Istio it is not very well documented, but is used in
the unit tests
(so quite stable). However even in this mode, Istio relies on a config store - file or an federated MCP-over-XDS server
- to store Istio-specific configs, endpoints, etc. 

