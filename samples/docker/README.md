# Running in a local docker container

This is intended for local development/debugging.

Note that the docker container will not be able to 
connect to the cloud unless a VPC is used. 

WIP to create a tunnel similar with 'ssh -R', and
use multi-network - which would allow more connectivity.

For now the this sample shows the docker image starting
and getting configs from ASM. It is possible to have use 
ServiceEntry and check connectivity to other reachable
workloads.


Unlike CloudRun, where we use the metadata server, this
requires:

- PROJECT_ID
- PROJECT_NUMBER
- CLUSTER_NAME
- CLUSTER_LOCATION
- WORKLOAD_NAMESPACE
- WORKLOAD_NAME

In addition a GSA file is needed.
