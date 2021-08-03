package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/costinm/krun/pkg/k8s"
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
    apiVersion: v1
    kind: Config
    clusters:
    - cluster:
        certificate-authority-data: {{ .ca }}
        server: https://{{ .server }}
      name: {{ .name }}
    contexts:
    - context:
        cluster: {{ .name }}
        user: {{ .name }} 
      name: {{ .name }}
    current-context: {{ .name }}
    preferences: {}
    users:
    - name: {{ .name }}
      user:
        auth-provider:
          name: gcp
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


	kr := &k8s.KRun{}
	//if meshID == "" {
	//  // if location is specified, create a single-cluster config.
	//	err := kr.CreateClusterConfig(gcpProj, location, cluster)
	//	if err != nil {
	//		panic(err)
	//	}
	//} else {
	//kr.AllHub(gcpProj, cluster, meshIDLabel, meshID)
	err := kr.AllClusters(gcpProj, "", "mesh_id", "")
	if err != nil {
		panic(err)
	}
	//}
	err = kr.SaveKubeConfig()
	if err != nil {
		panic(err)
	}

	tmpl := template.New("secret")
	tmpl, err = tmpl.Parse(IstioSecretTemplate)
	if err != nil {
		panic(err)
	}

	for _, c:= range kr.Clusters {
		buf := &bytes.Buffer{}
		cn := "gke_" + gcpProj + "_" + c.Location + "_" + c.Name
		cn = strings.ReplaceAll(cn, "_", "-")
		tmpl.Execute(buf, map[string]string{
			"name": cn,
			"ca": string(c.MasterAuth.ClusterCaCertificate),
			"server": c.Endpoint,
		})

		ioutil.WriteFile("secret-" + cn + ".yaml", buf.Bytes(), 0700)
	}
}

