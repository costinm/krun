package snigate

import (
	"os"
	"testing"
	"time"

	"github.com/costinm/krun/pkg/k8s"
)

func TestSNIGate(t *testing.T) {
	kr := k8s.New()
	_, err := InitSNIGate(kr, ":0", ":0")
	if err != nil {
		t.Fatal("Failed to connect to GKE ", time.Since(kr.StartTime), kr, os.Environ(), err)
	}


}
