package sts

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/costinm/cloud-run-mesh/pkg/gcp"
	"github.com/costinm/cloud-run-mesh/pkg/mesh"
)

// TestSTS uses a k8s connection and env to locate the mesh, and tests the token generation.
func TestSTS(t *testing.T) {
	kr := mesh.New("")

	ctx, cf := context.WithTimeout(context.Background(), 10*time.Second)
	defer cf()

	err := kr.LoadConfig(ctx)
	if err != nil {
		t.Skip("Failed to connect to GKE, missing kubeconfig ", time.Since(kr.StartTime), kr, os.Environ(), err)
	}

	if kr.ProjectNumber == "" {
		t.Skip("Skipping STS test, PROJECT_NUMBER required")
	}
	masterT, err := kr.GetToken(ctx, kr.TrustDomain)
	if err != nil {
		t.Fatal(err)
	}

	log.Println(mesh.TokenPayload(masterT), kr.ProjectNumber)

	s, err := NewSTS(kr)
	if err != nil {
		t.Fatal(err)
	}

	f, err := s.TokenFederated(ctx, masterT)
	if err != nil {
		t.Fatal(err)
	}

	a, err := s.TokenAccess(ctx, f, "")
	if err != nil {
		t.Fatal(err)
	}

	a, err = s.TokenAccess(ctx, f, "https://foo.bar")
	if err != nil {
		t.Fatal(err)
	}
	log.Println(mesh.TokenPayload(a))
}
