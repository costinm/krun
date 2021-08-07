# Istio gRPC support

Istio uses injection for proxyless gRPC support. 

Agent uses 2 env variables:
- GRPC_XDS_BOOTSTRAP - path to the to-be-generated bootstrap file
- DISABLE_ENVOY - Disables all Envoy agent features.

