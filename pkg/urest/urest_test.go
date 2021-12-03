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
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/costinm/krun/pkg/mesh"
	"gopkg.in/yaml.v2"
)

// Requires a GSA (either via GOOGLE_APPLICATION_CREDENTIALS, gcloud config, metadata) with hub and
// container access.
// Requires a kube config - the default cluster should be in same project.
//
// Will verify kube config loading and queries to hub and gke.
//
func TestUK8S(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	defer cf()

	uk := New()

	kcd, err := ioutil.ReadFile(os.Getenv("HOME") + "/.kube/config")
	if err != nil {
		t.Skip("No k8s config", err)
	}
	kc := &KubeConfig{}
	err = yaml.Unmarshal(kcd, kc)
	if err != nil {
		t.Fatal("Failed to parse k8c", err)
	}
	if kc.CurrentContext == "" {
		t.Fatal("No default context", kc)
	}
	ecl, _, err := extractClusters(uk, kc)
	if err != nil {
		t.Fatal("Failed to load k8s", err)
	}

	//
	uk.Current = ecl
	uk.ProjectID = ecl.ProjectId

	t.Run("kubeconfig", func(t *testing.T) {
		cm, err := uk.GetCM(ctx, "istio-system", "mesh-env")
		if err != nil {
			t.Fatal("Failed to load k8s", err)
		}
		log.Println(cm)
	})

	tok, err := uk.TokenProvider(ctx, "")

	t.Run("hublist", func(t *testing.T) {
		cd, err := getHubClusters(ctx, uk, tok, uk.ProjectID)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cd {
			//uk.Current = c

			log.Println(c)
		}
	})

	t.Run("gkelist", func(t *testing.T) {
		cd, err := getGKEClusters(ctx, uk, tok, uk.ProjectID)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cd {
			log.Println(c)
		}
	})

	t.Run("init", func(t *testing.T) {
		kr := mesh.New("")
		kr.ProjectId = uk.ProjectID

		_, err := K8SClient(ctx, kr)

		cd, err := getGKEClusters(context.TODO(), nil, uk.ProjectID, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cd {
			log.Println(c)
		}
	})

}
