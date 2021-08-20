# Using mesh without iptables

If krun starts as regular user, or runs in an environment where iptable config fails (no permission), it will fallback
to 'whitebox' mode, using HTTP_PROXY and local ports configured using Sidecar API when calling mesh services.

## HttpProxy and Sidecar

MeshConfig has a proxy_http_port setting - documented only as an option in the reference doc.

It has tests - 15007 seems to be used in tests, so we'll use this value. For Istio up to 1.11 you must set the 
global option to enable this mode. In 1.12, https://github.com/istio/istio/pull/34774 fixes this.

Main use in code is in listener.go buildHTTPProxy - which adds the extra listener and RDS combining all routes.

For TCP, Sidecar provides an API to redirect, with some limitations.


## Interception mode NONE

Set using sidecar.istio.io/interceptionMode: NONE which results in ISTIO_META_INTERCEPTION_MODE=none env variable.
This is required for Istiod to generate 'bindToPort=true' and related configs.

Example and test in istio/tests/testdata/config/none.yaml.


