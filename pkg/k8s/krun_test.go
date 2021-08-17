package k8s

import (
	"os"
	"testing"
	"time"
)

// Requires KUBECONFIG or $HOME/.kube/config
// The cluster must have MCP enabled.
// The test environment must have envoy in /usr/local/bin
func TestK8S(t *testing.T) {
	os.Mkdir("../../../out", 0775)
	os.Chdir("../../../out")

	kr := &KRun{
		StartTime: time.Now(),
	}

	err := kr.InitK8SClient()
	if err != nil {
		t.Fatal("Failed to connect to GKE ", time.Since(kr.StartTime), kr, os.Environ(), err)
	}

	kr.LoadConfig()

	kr.Refresh()

	kr.FindXDSAddr()

	kr.StartIstioAgent()

	t.Log(kr)

}

