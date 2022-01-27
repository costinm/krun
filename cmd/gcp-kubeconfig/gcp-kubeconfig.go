//Copyright 2021 Google LLC
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    https://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/costinm/krun/gcp"
	gcp2 "github.com/costinm/krun/k8s/gcp"
	"github.com/costinm/krun/pkg/mesh"
	"k8s.io/client-go/tools/clientcmd"
)

// IstioSecretTemplate generates a K8S Secret for GCP with istio annotations.
// Requires a metadata server or ADC.json.
//
// params: cluster name, k8s ca and server
const IstioSecretTemplate = `
apiVersion: v1
kind: Secret
metadata:
  annotations:
    networking.istio.io/cluster: {{ .name }}
  labels:
    istio/multiCluster: "true"
  name: istio-remote-secret-{{ .name }}
  namespace: istio-system
stringData:
  {{ .name }}: |
    {{ .kubeConfig }}
---
`

// Will create a kubeconfig and individual secrets, with all GKE and hub clusters.
//
func main() {
	gcpProj := os.Getenv("PROJECT_ID")
	//location := os.Getenv("LOCATION")
	//cluster := os.Getenv("CLUSTER")

	// Used to provide access to all clusteres in the mesh
	//meshID := os.Getenv("MESH_ID")

	kr := mesh.New()
	//if meshID == "" {
	//  // if location is specified, create a single-cluster config.
	//	err := kr.Mesh.CreateClusterConfig(gcpProj, location, cluster)
	//	if err != nil {
	//		panic(err)
	//	}
	//} else {
	//kr.AllHub(gcpProj, cluster, meshIDLabel, meshID)
	cl, err := gcp2.AllClusters(context.Background(), kr, "", "mesh_id", "")
	if err != nil {
		panic(err)
	}

	if len(cl) == 0 {
		log.Println("No clusters")
		return
	}
	kc := cl[0].KubeConfig
	for _, c := range cl {
		gcp.MergeKubeConfig(kc, c.KubeConfig)
	}
	err = gcp.SaveKubeConfig(kc, "", "config")
	if err != nil {
		panic(err)
	}

	tmpl := template.New("secret")
	tmpl, err = tmpl.Parse(IstioSecretTemplate)
	if err != nil {
		panic(err)
	}

	for _, c := range cl {
		buf := &bytes.Buffer{}
		cn := "gke_" + gcpProj + "_" + c.ClusterLocation + "_" + c.ClusterName
		cn = strings.ReplaceAll(cn, "_", "-")
		cfgjs, err := clientcmd.Write(*kc)
		if err != nil {
			continue
		}
		tmpl.Execute(buf, map[string]string{
			"name":       cn,
			"kubeConfig": string(cfgjs),
		})

		ioutil.WriteFile("secret-"+cn+".yaml", buf.Bytes(), 0700)
	}
}
