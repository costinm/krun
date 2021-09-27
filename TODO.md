Short note on remaining work, before the new repo is created for issue tracking:

[] P0: Only forward to registered services. Watch services with the CR label, use this to add the JWT.

[] P0: Add EnvoyFilter to handle the localhost forwarding. Need envoy listner for inbound to be patched to bind to port
and behave like a normal forwarder, and forward to this instead of 8080 ( so authz policy is applied )

[] P2: Move meshca public key to connector, save it to the mesh-env ( to avoid an extra roundtrip/complexity )

[] P1: connector to periodically refresh mesh-env, client to periodically read it (to handle root CA rotations, etc)

[] P1: load mesh-env from same namespace first, fallback to istio-system. Or read both and merge.

[] P0: tests for mesh-to-CR, more docs.

# Beta/public preview 

[] P0: add the code to wait for app and proxy ready before listening on port.

[] P1: port naming and type, multiple port support. Add a setting to allow other ports in the container to be exposed (mesh-env), 
   and indicate the type (TCP, H1, H2) - this is local to the container, not the same thing with the service type.

[] P2: examples for gvisor, without iptables (HTTP_PROXY + Sidecar)

[] P1: define labels on clusters to select, on Project to locate the hub if the clusters are not in same project
   This mprove auto-detectio. Fallback to other clusters, verify mesh-env is present. Add gateway address to mesh-env

[] P2(beta): replace CLUSTER_LOCATION/etc with only MESH_ENV, using the /projects/x/locations/y/clusters/z

[] P1: optimization: controller to generate a single config map with all the settings ( in create-if-doesn't-exist mode), to avoid
   reading 5 different things when auto-detecting. The key/value pairs should also be loaded from env, to ship even that lookup.
   The SHA(root cert) or full root certs can also be added.

[] P3: load additional labels, etc from the WorkloadGroup, using canonical service name as key.
       To avoid incresed latency, this should be done using the controller, creating a mesh-env configmap for 
       the service, and using the MESH_ENV to specify the config. No user-visible API needed.

[] P0/beta: custom in-cluster probers and tests to replace the curl tests and fortio 

[] P2: finish up proxyless gRPC generation and cert creation without using pilot-agent.

[] P2: K8S and GCP optional if the mesh-env is local and has a mesh connector (mesh connector will need to support STS, or cert must be installed)

[] P1/beta: controller to auto-register the SNI configs and related
    [] Revisions/services in same namespace, based on presence of the SA
    [] Control-plane side, called from CloudRun or pubsub, to create the SA.

[] P3: Add the hbone port, use instead of SNI routing ( has extra auth and meta)

# Cleanups/optimizations

[] P3: Use uK8S for the 1 or 2 calls needed at startup ( mesh-env, tokens ). Full k8s client 
is needed in connector, to create mesh-env and handle registration. Technically the client needs
a 'get' to any web server (storage bucket, etc) and may use an STS server for tokens instead of TokenRequest.

[] Support Fleet

[] Fix Anthos dashboard and metric labels
