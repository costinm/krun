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

package urest

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

// GCP secrets API
// Example:
//gcloud secrets create ca \
//--data-file <PATH-TO-SECRET-FILE> \
//--replication-policy automatic \
//--project dmeshgate \
//--format json \
//--quiet

const SecretsAPIURL = ""

type SecretsAPI struct {
}

func GcpSecret(ctx context.Context, uk *URest, token, p, n, v string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		"https://secretmanager.googleapis.com/v1/projects/"+p+"/secrets/"+n+
			"/versions/"+v+":access", nil)
	req.Header.Add("authorization", "Bearer "+token)

	res, err := uk.Client.Do(req)
	log.Println(res.StatusCode)
	rd, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var s struct {
		Payload struct {
			Data []byte
		}
	}
	err = json.Unmarshal(rd, &s)
	if err != nil {
		return nil, err
	}
	return s.Payload.Data, err
}

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
