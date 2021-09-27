## Running as non-root

KRun can also be used as regular user, howerver:

- iptables will not be set
- envoy (if found) will run with the current UID
- if envoy is not found, pilot-agent will still be started and generate proxyless gRPC config and certs
- all files will be created relative to current dir instead of root dir.
- Istio will use interception mode NONE - this enables 127.0.0.1:PORT bindings for mesh TCP services.

In this mode Istio can't capture traffic - it works in 'whitebox' mode, using HTTP_PROXY environment variable to capture
HTTP and Sidecar API for forwarding local ports to services.

It currently requires MeshConfig HttpProxyPort to be set - in 1.12 this will be automatically set for the workload, no
need for global config (PR#...)

Non-root mode is useful in Docker environments where iptables and/or running as root are not possible. For example
CI/CDs, current CloudRun VMs (minivm supports iptables), developer machine.

# Using mesh without iptables

If krun starts as regular user, or runs in an environment where iptable config fails (no permission), it will fallback
to 'whitebox' mode, using HTTP_PROXY and local ports configured using Sidecar API when calling mesh services.

## HttpProxy and Sidecar

MeshConfig has a proxy_http_port setting - documented only as an option in the reference doc.

It has tests - 15007 seems to be used in tests, so we'll use this value. For Istio up to 1.11 you must set the global
option to enable this mode. In 1.12, https://github.com/istio/istio/pull/34774 fixes this.

Main use in code is in listener.go buildHTTPProxy - which adds the extra listener and RDS combining all routes.

For TCP, Sidecar provides an API to redirect, with some limitations.

## Interception mode NONE

Set using sidecar.istio.io/interceptionMode: NONE which results in ISTIO_META_INTERCEPTION_MODE=none env variable. This
is required for Istiod to generate 'bindToPort=true' and related configs.

Example and test in istio/tests/testdata/config/none.yaml.


