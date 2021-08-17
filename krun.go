package main

import (
	"log"
	"os"
	"time"

	"github.com/costinm/krun/pkg/gcp"
	"github.com/costinm/hbone"
	"github.com/costinm/krun/pkg/k8s"
)


var initDebug func(run *k8s.KRun)

func main() {
	kr := &k8s.KRun{
		StartTime: time.Now(),
		VendorInit: gcp.InitGCP,
	}

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

	if kr.XDSAddr == "" {
		kr.FindXDSAddr()
	}

	if kr.XDSAddr != "-" {
		kr.StartIstioAgent()
	}

	kr.StartApp()


	if InitDebug != nil {
		// Split for conditional compilation (to compile without ssh dep)
		InitDebug(kr)
	}


	// TODO: wait for app and proxy ready
	if kr.XDSAddr != "-" {
		auth, err := hbone.LoadAuth("")
		if err != nil {
			log.Println("Failed to init hbone", err)
		}

		hb := hbone.New(auth)
		_, err = hbone.ListenAndServeTCP(":14009", hb.HandleAcceptedH2C)
		if err != nil {
			panic(err)
		}
	}
	select{}
}
