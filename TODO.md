Remaining work and ideas:

[] load labels, etc from the WorkloadGroup, using canonical service name as key.

[] zero config: get location, project from metadata server. Pick a cluster with mesh_id label
in same location ('istio-' prefix preferred ). Get XDS_ADDR, meta from mutating webhook.

[] P3: conditional compile for the gcp library (to evaluate size impact)
