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
)

func (kr *KRun) LoadConfig(ctx context.Context) error {
	// It is possible to have only one of the 2 mesh connector services installed
	if kr.XDSAddr == "" || kr.ProjectNumber == "" ||
		(kr.MeshConnectorAddr == "" && kr.MeshConnectorInternalAddr == "") {
		err := kr.loadMeshEnv(ctx)
		if err != nil {
			return err
		}
		// Adjust 'derived' values if needed.
		if kr.TrustDomain == "" && kr.ProjectId != "" {
			kr.TrustDomain = kr.ProjectId + ".svc.id.goog"
		}
	}

	return nil
}
