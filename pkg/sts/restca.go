package sts

// REST based interface with the CAs - to keep the binary size small.
// We just need to make 1 request at startup and maybe one per hour.

var (
	// access token for the p4sa.
	// Exchanged k8s token to p4sa access token.
	meshcaEndpoint = "https://meshca.googleapis.com:443/google.security.meshca.v1.MeshCertificateService/CreateCertificate"

	// JWT token with istio-ca or gke trust domain
	istiocaEndpoint = "/istio.v1.auth.IstioCertificateService/CreateCertificate"
)

// JWT tokens have audience https://SNI_NAME/istio.v1.auth.IstioCertificateService
// However for Istiod we should use 'istio-ca' or trustdomain.
//
// Headers:
// - te: trailers
// - content-type: application/grpc
// - grpc-previous-rpc-attempts
// - grpc-timeout
// - grpc-tags-bin, grpc-trace-bin
// -
func MeshCAClient() {

}
