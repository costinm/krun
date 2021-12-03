package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/costinm/krun/pkg/mesh"
	"github.com/costinm/krun/pkg/sts"
	"github.com/costinm/krun/pkg/uk8s"
)

var (
	aud       = flag.String("audience", "", "Audience, if empty an access token returned")
	provider  = flag.String("iss", "", "Issuer. Default is k8s")
	namespace = flag.String("n", "", "Namespace")
	sa        = flag.String("sa", "", "Service account")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	kr := mesh.New("")
	_, err := urest.K8SClient(ctx, kr)
	err = kr.LoadConfig(ctx)
	if err != nil {
		panic(err)
	}

	tokenProvider, err := sts.NewSTS(kr)

	if kr.MeshConnectorAddr == "" {
		log.Fatal("Failed to find in-cluster, missing 'hgate' service in mesh env")
	}

	kr.XDSAddr = kr.MeshConnectorAddr + ":15012"

	t, err := tokenProvider.GetRequestMetadata(ctx, *aud)
	if err != nil {
		log.Fatal("Failed to get token", err)
	}
	fmt.Println(t)
}
