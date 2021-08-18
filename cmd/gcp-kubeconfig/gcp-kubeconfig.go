package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/costinm/krun/pkg/gcp"
	"github.com/costinm/krun/pkg/k8s"
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


	kr := &k8s.KRun{}
	//if meshID == "" {
	//  // if location is specified, create a single-cluster config.
	//	err := kr.CreateClusterConfig(gcpProj, location, cluster)
	//	if err != nil {
	//		panic(err)
	//	}
	//} else {
	//kr.AllHub(gcpProj, cluster, meshIDLabel, meshID)
	cl, err := gcp.AllClusters(context.Background(), kr, "", "mesh_id", "")
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

	for _, c:= range cl {
		buf := &bytes.Buffer{}
		cn := "gke_" + gcpProj + "_" + c.ClusterLocation + "_" + c.ClusterName
		cn = strings.ReplaceAll(cn, "_", "-")
		cfgjs, err := clientcmd.Write(*kc)
		if err != nil {
			continue
		}
		tmpl.Execute(buf, map[string]string{
			"name": cn,
			"kubeConfig": string(cfgjs),
		})

		ioutil.WriteFile("secret-" + cn + ".yaml", buf.Bytes(), 0700)
	}
}

