Remaining work and ideas:

[] Add sample for the istio cluster configuration, with ACME for istiod.

[] Default namespace to the GSA, canonical service to the KNative service, project/region 
   from metadata server, and cluster name to 'istio'. Users will set a 'istio' cluster per
   region for the control plane configs and istiod (or MCP), rest of the clusters can use 
   the 'istio' cluster control plane ( auto-config multi-cluster )

[] Finish implementing mounts of secrets, configmaps. This will allow support for self-signed istiod

[] load labels, etc from the WorkloadGroup, using canonical service name as key.

[] zero config: get location, project from metadata server. Pick a cluster with mesh_id label
in same location ('istio-' prefix preferred ). Get XDS_ADDR, meta from mutating webhook.

[] P3: conditional compile for the gcp library (to evaluate size impact)
