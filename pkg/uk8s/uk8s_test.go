package uk8s

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"golang.org/x/oauth2/google"
	"gopkg.in/yaml.v2"
)

func TestUK8S(t *testing.T) {

	kcd, err := ioutil.ReadFile(os.Getenv("HOME") + "/.kube/config")
	if err != nil {
		t.Skip("No k8s config", err)
	}
	kc := &KubeConfig{}
	err = yaml.Unmarshal(kcd, kc)
	if err != nil {
		t.Fatal("Failed to parse k8c", err)
	}
	if kc.CurrentContext == "" {
		t.Fatal("No default context", kc)
	}
	uk, err := NewUK8S(kc)
	if err != nil {
		t.Fatal("Failed to load k8s", err)
	}

	me, err := uk.GetConfigMap("istio-system", "mesh-env")
	log.Println(me, err)

	me1, err := uk.GetConfigMap("istio-system", "istio")
	log.Println(me1, err)

	cd, err := GetContainers(context.TODO(), uk.ProjectID)
	log.Println(string(cd),err)
}

func GetContainers(ctx context.Context, p string) ([]byte, error){
	httpClient, err := google.DefaultClient(ctx,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Get("https://container.googleapis.com/v1/projects/" + p + "/locations/-/clusters")
	log.Println(res.StatusCode)
	rd, err := ioutil.ReadAll(res.Body)
	return rd, err

}

