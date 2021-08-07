package main

import (
	"context"
	"fmt"
	"os"

	k8s "github.com/costinm/krun/pkg/k8s"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Minimal tool to get a K8S token with audience.
func main() {
	ns := conf("NS", "default")
	ksa := conf("KSA", "default")
	aud := conf("AUD", "api")
	if len(os.Args) > 1 {
		aud = os.Args[1]
	}

	kr := &k8s.KRun{}
	err := kr.InitK8SClient()
	if err != nil {
		panic(err)
	}
	treq := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences: []string{aud},
		},
	}
	if err != nil {
		panic(err)
	}
	ts, err := kr.Client.CoreV1().ServiceAccounts(ns).CreateToken(context.Background(),
		ksa, treq, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Println(ts.Status.Token)
}

func conf(key, def string) string {
	r := os.Getenv(key)
	if r == "" {
		return def
	}
	return r
}

