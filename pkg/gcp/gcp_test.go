package gcp

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/costinm/cloud-run-mesh/pkg/k8s"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Requires GOOGLE_APPLICATION_CREDENTIALS or KUBECONFIG or $HOME/.kube/config
func TestK8S(t *testing.T) {
	os.Mkdir("../../out", 0775)
	os.Chdir("../../out")

	// ADC or runner having permissions are required
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		t.Skip("Missing PROJECT_ID")
		return
	}
	// For the entire test
	ctx, cf := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cf()

	kr := k8s.New()
	cl, err := AllClusters(ctx, kr, "", "mesh_id", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(cl) == 0 {
		t.Fatal("No ASM clusters")
	}
	if kr.ProjectId != projectID {
		t.Error("Project ID initialization", kr.ProjectId, projectID)
	}
	testCluster := cl[0]

	// Run the tests on the first found cluster, unless the test is run with env variables to select a specific
	// location and cluster name.

	t.Run("all_clusters", func(t *testing.T) {
		cl, err := AllClusters(ctx, kr, "", "", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(cl) == 0 {
			t.Fatal("No clusters")
		}
	})

	// WIP: using the hub, for multi-project
	t.Run("hub", func(t *testing.T) {
		kchub, err := AllHub(ctx, kr)
		if err != nil {
			t.Fatal(err)
		}
		for _, kh := range kchub {
			t.Log("Hub:", kh.ClusterName, kh.ClusterLocation)
		}
		if len(kchub) == 0 {
			t.Skip("No hub clusters registered")
		}

		c0 := kchub[0]
		rc, err := restConfig(c0.KubeConfig)
		if err != nil {
			t.Fatal(err)
		}
		client, err := kubernetes.NewForConfig(rc)
		if err != nil {
			t.Fatal(err)
		}
		err = checkClient(client)
		if err != nil {
			if se, ok := err.(*errors.StatusError); ok {
				if se.Status().Reason == "Forbidden" {
					t.Skip("GKE Connect not authorized")
				}
			}
			t.Fatal(err)
		}

	})

	t.Run("gke", func(t *testing.T) {
		// This is the main function for the package - given a KRun object, initialize the K8S Client based
		// on settings and GKE API result.
		kr1 := k8s.New()
		kr1.ProjectId = kr.ProjectId
		kr1.ClusterName = testCluster.ClusterName
		kr1.ClusterLocation = testCluster.ClusterLocation

		err = InitGCP(context.Background(), kr1)
		if err != nil {
			t.Fatal(err)
		}
		if kr1.Client == nil {
			t.Fatal("No client")
		}

		err = checkClient(kr1.Client)
		if err != nil {
			t.Fatal(err)
		}

	})
}

func checkClient(kc *kubernetes.Clientset) error {
	v, err := kc.ServerVersion() // /version on the server
	if err != nil {
		return err
	}
	log.Println("Cluster version", v)

	_, err = kc.CoreV1().ConfigMaps("istio-system").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	return nil
}
