package k8s

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"strings"

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
			os.Chown(path + "/" + k, 1337, 1337)
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

// FindXDSAddr will try to find the XDSAddr using in-cluster info.
func (kr *KRun) FindXDSAddr() {
	// TODO: find default tag, label, etc.
	// Current code is written for MCP, use XDS_ADDR explicitly
	// otherwise.
	s, err :=  kr.Client.CoreV1().ConfigMaps("istio-system").Get(context.Background(),
		"istio-asm-managed", metav1.GetOptions{})
	if err != nil {
		//
		panic(err)
	}
	meshCfg := s.Data["mesh"]
	kr.XDSAddr = meshCfgGet(meshCfg, "discoveryAddress")
	kr.MCPAddr = meshCfgGet(meshCfg, "ISTIO_META_CLOUDRUN_ADDR")
}

// quick qay to extract the value of a key from mesh config, without fully decoding it.
func meshCfgGet(meshCfg string, key string) string {
	start1 := strings.Index(meshCfg, key)
	val := meshCfg[start1+len(key)+1:]
	s0 := strings.Index(val, "\"")
	e0 := strings.Index(val[s0+1:], "\"")
	mcpAddr := val[s0+1: s0+e0+1]
	return mcpAddr
}
