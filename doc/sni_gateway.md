# SNI gateway

Communication from Pods to Krun-based workloads use the same mechanism as Istio multi-network.

A SNI gateway is deployed, and the service is 'attached' to the target network.

For reference: pilot/pkg/model/network.go

The multi-network gateways are loaded from the static config file or from k8s. The main criteria in k8s is having a
label
'topology.istio.io/network' with the value being the network name. We'll use 'hb' as network.

networking.istio.io/gatewayPort can be used to override the default port 15443 - we're not using this.

## SNI routes

Istio generates clusters with:

```
  "sni": "outbound_.8080_._.fortio.fortio.svc.cluster.local"
```

The SNI gateway currently also supports the 'natural' format, without mangling:

``` 
  fortio.fortio.svc.cluster.local
```

