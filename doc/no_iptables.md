# Using mesh without iptables

## HttpProxy and Sidecar

MeshConfig has a proxy_http_port setting - documented only as an option.

It has tests - 15007 seems to be used in tests.

Main use is in listener.go buildHTTPProxy - which adds the extra listener.

For TCP, Sidecar provides an API to redirect, with some limitations.


## Interception mode NONE

Set using sidecar.istio.io/interceptionMode: NONE which results 
in ISTIO_META_INTERCEPTION_MODE=none env variable.



