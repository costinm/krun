package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/costinm/krun/pkg/hbone"
	"github.com/costinm/krun/pkg/k8s"
)

// Based on krun, start pilot-agent to get the
// certs and create the XDS proxy, and implement
// a SNI to H2 proxy - similar with the current multi-net gateway
// protocol from Istio side.
func main() {
	kr := &k8s.KRun{
		StartTime: time.Now(),
	}
	kr.LoadConfig()

	err := kr.InitK8SClient()
	if err != nil {
		log.Fatal("Failed to connect to GKE ", time.Since(kr.StartTime), kr, os.Environ(), err)
	}

	kr.Refresh()

	if kr.XDSAddr == "" {
		kr.FindXDSAddr()
	}

	if kr.XDSAddr != "-" {
		proxyConfig := fmt.Sprintf(`{"discoveryAddress": "%s"}`, kr.XDSAddr)
		kr.ExtraEnv = []string{"GRPC_XDS_BOOTSTRAP=./var/run/grpc.json"}
		kr.StartIstioAgent(proxyConfig)
	}

	auth := &hbone.Auth{}
	auth.InitKeys()

	h2r := hbone.H2RServer{Auth: auth}

	h2r.Start()

	select{}

}
