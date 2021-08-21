# Istio gRPC support

Istio uses injection for proxyless gRPC support. 

Agent uses 2 env variables:
- GRPC_XDS_BOOTSTRAP - path to the to-be-generated bootstrap file. To simplify UX, this is set automatically.
- DISABLE_ENVOY - Disables all Envoy agent features. Krun automatically sets this if envoy is not detected in the image.


