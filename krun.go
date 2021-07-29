package main

import (
	"fmt"
	"log"
	"os"

	"github.com/costinm/krun/pkg/hbone"
	"github.com/costinm/krun/pkg/k8s"
)


var initDebug func(run *k8s.KRun)

func main() {
	kr := &k8s.KRun{}
	kr.InitromEnv()

	k8sClient, err := kr.GetK8S()
	if err != nil {
		panic(err)
	}

	if len(os.Args) == 1 {
		// Default gateway label for now, we can customize with env variables.
		kr.Gateway = "ingressgateway"
		log.Println("Starting in gateway mode", os.Args)
	}

	kr.Client = k8sClient

	kr.Refresh()

	xdsAddr := os.Getenv("XDS_ADDR")
	if xdsAddr != "" {
		proxyConfig := fmt.Sprintf(`{"discoveryAddress": "%s"}`, xdsAddr)
		kr.StartIstioAgent(proxyConfig)
	}

	kr.StartApp()


	if InitDebug != nil {
		// Split for conditional compilation (to compile without ssh dep)
		InitDebug(kr)
	}


	// TODO: wait for app and proxy ready
	hb := &hbone.HBone{
	}
	err = hb.Init()
	if err != nil {
		panic(err)
	}

	err = hb.Start(":14009")
	if err != nil {
		panic(err)
	}

	select{}
}
