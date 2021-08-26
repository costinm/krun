package main

import (
	"context"
	"flag"
	"fmt"

	_ "github.com/costinm/cloud-run-mesh/pkg/gcp"
	k8s "github.com/costinm/cloud-run-mesh/pkg/k8s"
)

var (
	nsFlag  = flag.String("ns", "default", "namespace")
	ksaFlag = flag.String("ksa", "default", "kubernetes service account")
)

// Minimal tool to get a K8S token with audience.
func main() {
	flag.Parse()
	aud := "api"
	if len(flag.Args()) > 1 {
		aud = flag.Args()[0]
	}

	kr := k8s.New()
	if kr.Namespace == "" {
		kr.Namespace = *nsFlag
	}
	if kr.KSA == "" {
		kr.KSA = *ksaFlag
	}
	err := kr.InitK8SClient(context.Background())
	if err != nil {
		panic(err)
	}

	tok, err := kr.GetToken(context.Background(), aud)
	if err != nil {
		panic(err)
	}

	fmt.Println(tok)
}
