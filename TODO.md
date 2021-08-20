Remaining work and ideas:

[] P0: figure out why intercepting all outbound traffic results in restart on 
CloudRun. Add the ip ranges from GKE cluster info.

[] P0: if running agent as root ( for DNS ), it is no longer excluded from iptables
interception. Try running a GID 1337, otherwise DNS proxy will need to be refactored
or the socket for port 53 be passed via UDS or other mean.

[] P0: add the code to wait for app and proxy ready before listening on port.

[] P1: examples for gvisor, without iptables (HTTP_PROXY + Sidecar)

[] P1: samples with proxyless gRPC, send PR to fortio to add support.

[] P2: auto-configure the cluster, by listing all clusters in same region with same
mesh ID, and picking any cluster that works, using MutatingWebhook to find istiod.
The clusters in the mesh are supposed to be equivalent. Eventually fallback to other
regions, for higher availability.

[] Add sample for the istio cluster configuration - ACME certs and gateway for istiod.

[] Default namespace to the GSA, canonical service to the KNative service, project/region 
   from metadata server, and cluster name to 'istio'. Users will set a 'istio' cluster per
   region for the control plane configs and istiod (or MCP), rest of the clusters can use 
   the 'istio' cluster control plane ( auto-config multi-cluster )

[] Finish implementing mounts of secrets, configmaps. This will allow support for self-signed istiod

[] load labels, etc from the WorkloadGroup, using canonical service name as key.

[] zero config: get location, project from metadata server. Pick a cluster with mesh_id label
in same location ('istio-' prefix preferred ). Get XDS_ADDR, meta from mutating webhook.

[] P3: conditional compile for the gcp library (to evaluate size impact)
