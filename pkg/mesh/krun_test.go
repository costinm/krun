package mesh

import (
	context2 "context"
	"os"
	"testing"
	"time"

	// Required for k8s client to link in the authenticator
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// Requires KUBECONFIG or $HOME/.kube/config
// The cluster must have MCP enabled.
// The test environment must have envoy in /usr/local/bin
func TestK8S(t *testing.T) {
	os.Mkdir("../../../out", 0775)
	os.Chdir("../../../out")

	kr := New("")

	err := kr.LoadConfig(context2.Background())
	if err != nil {
		t.Skip("Failed to connect to GKE, missing kubeconfig ", time.Since(kr.StartTime), kr, os.Environ(), err)
	}

	// For Istio agent
	kr.RefreshAndSaveFiles()

	kr.StartIstioAgent()

	t.Log(kr)

}
