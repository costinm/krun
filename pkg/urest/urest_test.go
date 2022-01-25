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

package urest_test

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/costinm/krun/pkg/mesh"
	"github.com/costinm/krun/pkg/urest"
	"golang.org/x/oauth2/google"
	"gopkg.in/yaml.v2"
)

// Requires a GSA (either via GOOGLE_APPLICATION_CREDENTIALS, gcloud config, metadata) with hub and
// container access.
// Requires a kube config - the default cluster should be in same project.
//
// Will verify kube config loading and queries to hub and gke.
//
func TestURest(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	defer cf()

	uk := urest.New()
	ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		t.Fatal(err)
	}
	uk.TokenProvider = func(ctx context.Context, s string) (string, error) {
		t, err := ts.Token()
		if err != nil {
			return "", err
		}
		return t.AccessToken, nil
	}
	kcd, err := ioutil.ReadFile(os.Getenv("HOME") + "/.kube/config")
	if err != nil {
		t.Skip("No k8s config", err)
	}
	kc := &urest.KubeConfig{}
	err = yaml.Unmarshal(kcd, kc)
	if err != nil {
		t.Fatal("Failed to parse k8c", err)
	}
	if kc.CurrentContext == "" {
		t.Fatal("No default context", kc)
	}

	ecl, _, err := urest.KubeConfig2RestCluster(uk, kc)
	if err != nil {
		t.Fatal("Failed to load k8s", err)
	}

	//
	uk.Current = ecl
	uk.ProjectID = ecl.ProjectId

	// Cluster must have mesh connector installed
	t.Run("kubeconfig", func(t *testing.T) {
		cm, err := uk.GetCM(ctx, "istio-system", "mesh-env")
		if err != nil {
			t.Fatal("Failed to load k8s", err)
		}
		log.Println(cm)
	})

	// Access tokens
	tok, err := uk.TokenProvider(ctx, "")

	t.Run("hublist", func(t *testing.T) {
		cd, err := urest.Hub2RestClusters(ctx, uk, tok, uk.ProjectID)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cd {
			//uk.Current = c

			log.Println(c)
		}
	})

	t.Run("gkelist", func(t *testing.T) {
		cd, err := urest.GKE2RestCluster(ctx, uk, tok, uk.ProjectID)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cd {
			log.Println(c)
		}
	})

	t.Run("secret", func(t *testing.T) {
		cd, err := urest.GcpSecret(ctx, uk, tok, uk.ProjectID, "ca", "1")
		if err != nil {
			t.Fatal(err)
		}
		log.Println(string(cd))
	})

	t.Run("init", func(t *testing.T) {
		kr := mesh.New("")
		kr.ProjectId = uk.ProjectID

		uk1, err := urest.K8SClient(ctx, kr)
		uk1.Client = http.DefaultClient

		cd, err := urest.GKE2RestCluster(context.TODO(), uk1, uk.ProjectID, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cd {
			log.Println(c)
		}
	})

}
