# Proxyless gRPC support

In-cluster Istio uses injection for proxyless gRPC support. The KRun project automatically generates the bootstrap file
and sets GRPC_XDS_BOOTSTRAP env variable - any gRPC application that has XDS integration compiled in can use it directly. 

The only setting for the user is:
- DISABLE_ENVOY - Disables all Envoy agent features. Krun will also automatically sets this if envoy is not detected
  in the image.




