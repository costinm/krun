package sts

import (
	"context"
	"strings"

	"github.com/costinm/krun/pkg/mesh"
)

// K8SCredentials returns tokens for Istiod.
// They have trust domain as audience
type K8SCredentials struct {
	KRun *mesh.KRun

	// If set, the audience will be used instead of the one from the request.
	// Used for Citadel, XDS - where "istio-ca" or "trust domain" can be used.
	Audience string
}

// RequireTranportSecurity is part of gRPC interface, returning false because we also support secure networks (low-level)
func (istiodTP *K8SCredentials) RequireTransportSecurity() bool {
	return false
}

// GetRequestMetadata implements credentials.PerRPCCredentials, specifically for 'trustDomain' tokens used by
// Istiod. Audience example: https://istiod.istio-system.svc/istio.v1.auth.IstioCertificateService (based on SNI name!)
func (istiodTP *K8SCredentials) GetRequestMetadata(ctx context.Context, aud ...string) (map[string]string, error) {
	a := aud[0]
	if len(aud) > 0 && strings.Contains(aud[0], "/istio.v1.auth.IstioCertificateService") {
		//a = "istio-ca"
		a = istiodTP.KRun.TrustDomain
	}
	if istiodTP.Audience != "" {
		a = istiodTP.Audience // override
	}
	// TODO: same for the XDS stream

	kt, err := istiodTP.KRun.GetToken(ctx, a)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"authorization": "Bearer " + kt,
	}, nil
}

