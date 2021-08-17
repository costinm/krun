package main

import (
	"log"
	"os"
	"time"

	"github.com/costinm/hbone"
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

	err := kr.InitK8SClient()
	if err != nil {
		log.Fatal("Failed to connect to GKE ", time.Since(kr.StartTime), kr, os.Environ(), err)
	}

	kr.LoadConfig()
	kr.Refresh()

	auth, err := hbone.LoadAuth("")
	if err != nil {
		log.Fatal("Failed to load certificates", err)
	}

	h2r := hbone.New(auth)

	_, err = hbone.ListenAndServeTCP(":15443", h2r.HandleSNIConn)
	if err != nil {
		log.Fatal("Failed to start SNI tunnel", err)
	}

	_, err = hbone.ListenAndServeTCP(":15442", h2r.HandleH2RSNIConn)
	if err != nil {
		log.Fatal("Failed to start H2R SNI tunnel ", err)
	}

	_, err = hbone.ListenAndServeTCP(":15441", h2r.HandleH2RSNIConn)
	if err != nil {
		log.Fatal("Failed to start H2R tunnel", err)
	}

	select{}

}
