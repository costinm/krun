Short note on remaining work, before the new repo is created for issue tracking:

[] P0: add the code to wait for app and proxy ready before listening on port.

[] P1: examples for gvisor, without iptables (HTTP_PROXY + Sidecar)

[] P1: samples with proxyless gRPC, send PR to fortio to add support.

[] P1: define labels on clusters to select, on Project to locate the hub if the clusters are not in same project

[] P2: replace CLUSTER_LOCATION/etc with only CLUSTER_NAME, using the /projects/x/locations/y/clusters/z

[] P0: controller to auto-register the SNI configs and related 

[] P1: optimization: controller to generate a single config map with all the settings ( in create-if-doesn't-exist mode), to avoid
   reading 5 different things when auto-detecting. The key/value pairs should also be loaded from env, to ship even that lookup.
   The SHA(root cert) or full root certs can also be added.

[] P3: Revert the creation of secrets/configmaps - use konfigure instead ( link it as a library )

[] P3: load additional labels, etc from the WorkloadGroup, using canonical service name as key.

[] P0: custom in-cluster probers and tests to replace the curl tests and fortio 

[] P1: finish up proxyless gRPC generation and cert creation without using pilot-agent.

[] Improve auto-detection - define labels to look for, fallback to other clusters, verify mesh-env is present. Add gateway address to mesh-env

[] K8S and GCP optional if the mesh-env is local and has a mesh connector (mesh connector will need to support STS, or cert must be installed)


# Mesh connector 

[] Register
[] Only forward to registered service
[] Add the hbone port, use instead of SNI routing ( has extra auth and meta)
[] Figure out why using the egress is not working.
