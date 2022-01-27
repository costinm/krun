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

func GcpSecret(ctx context.Context, uk *UK8S, token, p, n, v string) ([]byte, error) {
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
