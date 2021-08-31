package sts

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/costinm/cloud-run-mesh/pkg/gcp"
	"github.com/costinm/cloud-run-mesh/pkg/k8s"
)

func TestSTS(t *testing.T) {
	kr := k8s.New()

	ctx, cf := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cf()

	err := kr.InitK8SClient(ctx)
	if err != nil {
		t.Skip("Failed to connect to GKE, missing kubeconfig ", time.Since(kr.StartTime), kr, os.Environ(), err)
	}

	kr.LoadConfig()

	kr.Refresh()

	// Has the side-effect of loading the project number
	kr.FindXDSAddr()

	masterT, err := kr.GetToken(ctx, kr.TrustDomain)
	if err != nil {
		t.Fatal(err)
	}

	log.Println(k8s.TokenPayload(masterT), kr.ProjectNumber)

	s, err  := NewSTS(kr)
	if err != nil {
		t.Fatal(err)
	}

	f, err := s.TokenFederated(ctx, masterT)
	if err != nil {
		t.Fatal(err)
	}
	log.Println(f)

	a, err := s.TokenAccess(ctx, f, "")
	if err != nil {
		t.Fatal(err)
	}
	log.Println(a)

	a, err = s.TokenAccess(ctx, f, "https://foo.bar")
	if err != nil {
		t.Fatal(err)
	}
	log.Println(k8s.TokenPayload(a))
}
