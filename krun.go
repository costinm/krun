package main

import (
	"log"
	"os"
	"time"

	"github.com/costinm/hbone"
	"github.com/costinm/krun/pkg/gcp"
	"github.com/costinm/krun/pkg/k8s"
)


var initDebug func(run *k8s.KRun)

func main() {
	kr := k8s.New()

	kr.VendorInit = gcp.InitGCP

	err := kr.InitK8SClient()
	if err != nil {
		log.Fatal("Failed to connect to GKE ", time.Since(kr.StartTime), kr, os.Environ(), err)
	}

	kr.LoadConfig()

	if len(os.Args) == 1 {
		// Default gateway label for now, we can customize with env variables.
		kr.Gateway = "ingressgateway"
		log.Println("Starting in gateway mode", os.Args)
	}

	kr.Refresh()

	meshMode := true

	if _, err := os.Stat("/usr/local/bin/pilot-agent"); os.IsNotExist(err) {
		meshMode = false
	}
	if kr.XDSAddr == "-" {
		meshMode = false
	}

	if meshMode {
		// Use k8s client to autoconfigure, reading from cluster.
		if kr.XDSAddr == "" {
			err = kr.FindXDSAddr()
			if err != nil {
				log.Fatal("Failed to locate the XDS server ", err)
			}
		}

		if kr.XDSAddr != "-" {
			err := kr.StartIstioAgent()
			if err != nil {
				log.Fatal("Failed to start the mesh agent")
			}
		}
		// TODO: wait for proxy ready before starting app.
	}


	kr.StartApp()

	// Start internal SSH server, for debug and port forwarding. Can be conditionally compiled.
	if initDebug != nil {
		// Split for conditional compilation (to compile without ssh dep)
		initDebug(kr)
	}


	// TODO: wait for app and proxy ready

	if meshMode {
		auth, err := hbone.LoadAuth("")
		if err != nil {
			log.Fatal("Failed to find mesh certificates", err)
		}
		// TODO: use an in-cluster secret or self-signed certs if mesh mode disabled

		hb := hbone.New(auth)
		_, err = hbone.ListenAndServeTCP(":15009", hb.HandleAcceptedH2C)
		if err != nil {
			panic(err)
		}

		// TODO: if east-west gateway present, create a connection.
	}
	select{}
}
