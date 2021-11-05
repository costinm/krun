// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mesh

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
)

// K8SCredentials returns tokens for Istiod.
// They have trust domain as audience
type K8SCredentials struct {
	KRun *KRun

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

// RequireTranportSecurity is part of gRPC interface, returning false because we also support secure networks (low-level)
func (kr *KRun) RequireTransportSecurity() bool {
	return false
}

// GetRequestMetadata implements credentials.PerRPCCredentials with normal audience semantics, returning tokens signed
// by K8S APIserver. For GCP tokens, use 'sts' package.
func (kr *KRun) GetRequestMetadata(ctx context.Context, aud ...string) (map[string]string, error) {
	a0 := ""
	if len(aud) > 0 {
		a0 = aud[0]
	}
	if len(aud) > 1 {
		return nil, errors.New("Single audience supporte")
	}
	kt, err := kr.GetToken(ctx, a0)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"authorization": "Bearer " + kt,
	}, nil
}

// GetToken returns a token with the given audience for the current KSA, using CreateToken request.
// Used by the STS token exchanger.
func (kr *KRun) GetToken(ctx context.Context, aud string) (string, error) {
	return kr.TokenProvider.GetToken(ctx, aud)
}

// TokenPayload returns the decoded token. Used for logging/debugging token content, without printing the signature.
func TokenPayload(jwt string) string {
	jwtSplit := strings.Split(jwt, ".")
	if len(jwtSplit) != 3 {
		return ""
	}
	//azp,"email","exp":1629832319,"iss":"https://accounts.google.com","sub":"1118295...
	payload := jwtSplit[1]

	payloadBytes, err := base64.RawStdEncoding.DecodeString(payload)
	if err != nil {
		return ""
	}

	return string(payloadBytes)
}
