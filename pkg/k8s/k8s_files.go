package k8s

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func initSecret(k8sClient *kubernetes.Clientset,  ns string, name string, path string) {
	s, err := k8sClient.CoreV1().Secrets(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	for k, v := range s.Data {
		err = ioutil.WriteFile(path + "/" + k, v, 0700)
		if os.Getuid() == 0 {
			_ = os.Chown(path + "/" + k, 1337, 1337)
		}
		if err != nil {
			log.Println("Failed to init secret ", name, path, k, err)
		}
	}
}

func initCM(k8sClient *kubernetes.Clientset,  ns string, name string, path string) {
	s, err := k8sClient.CoreV1().ConfigMaps(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	for k, v := range s.Data {
		err = ioutil.WriteFile(path + "/" + k, []byte(v), 0755)
		if err != nil {
			log.Println("Failed to init secret ", name, path, k, err)
		}
	}
}


func (kr *KRun) GetCM(ctx context.Context, ns string, name string) (map[string]string, error) {
	s, err := kr.Client.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return map[string]string{}, err
	}

	return s.Data, nil
}

func (kr *KRun) GetSecret(ctx context.Context, ns string, name string) (map[string][]byte, error) {
	s, err := kr.Client.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return map[string][]byte{}, err
	}

	return s.Data, nil
}
