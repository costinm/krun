package k8s

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (kr *KRun) Refresh() {
	for aud, f := range kr.Aud2File {
		InitToken(kr.Client, kr.Namespace, kr.KSA, aud, f)
	}
	for k, v := range kr.Secrets2Dirs {
		initSecret(kr.Client, kr.Namespace, k, v)
	}
	for k, v := range kr.CM2Dirs {
		initCM(kr.Client, kr.Namespace, k, v)
	}

	time.AfterFunc(30 * time.Minute, kr.Refresh)
}

func InitToken(client *kubernetes.Clientset, ns string, ksa string, audience string, s2 string) error {
	treq := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences: []string{audience},
		},
	}
	ts, err := client.CoreV1().ServiceAccounts(ns).CreateToken(context.Background(),
		ksa, treq, metav1.CreateOptions{})
	if err != nil {
		log.Println("Error creating ", ns, ksa, audience, err)
		return err
	}

	lastSlash := strings.LastIndex(s2, "/")
	err = os.MkdirAll(s2[:lastSlash], 0755)
	if err != nil {
		log.Println("Error creating dir", ns, ksa, s2[:lastSlash])
	}
	// Save the token, readable by app. Little value to have istio token as different user,
	// for this separate container/sandbox is needed.
	err = ioutil.WriteFile(s2, []byte(ts.Status.Token), 0644)
	if err != nil {
		log.Println("Error creating ", ns, ksa, audience, s2, err)
		return err
	}

	return nil
}
