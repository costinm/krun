# SNI gateway

Communication from Pods to Krun-based workloads use the same mechanism as Istio multi-network. 

A SNI gateway is deployed, and the service is 'attached' to the target network.

For reference: pilot/pkg/model/network.go 

The multi-network gateways are loaded from the static config file or from k8s. The main criteria is having a label
'topology.istio.io/network' with the value beeing the network name. We'll use 'hb' as network.

networking.istio.io/gatewayPort can be used to override the default port 15443 - we're not using this.
