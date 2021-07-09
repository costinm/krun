package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/costinm/krun/pkg/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var initDebug func(run *k8s.KRun)


func main() {
	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		ns = "default"
	}
	ksa := os.Getenv("SERVICE_ACCOUNT")
	if ksa == "" {
		ksa = "default"
	}
	name := os.Getenv("LABEL_APP")
	if name == "" {
		name = "default"
	}

	k8sClient, err := k8s.GetK8S()
	if err != nil {
		panic(err)
	}

	kr := &k8s.KRun{
		Name: name,
		Namespace: ns,
	}

	if len(os.Args) == 1 {
		// Default gateway label for now, we can customize with env variables.
		kr.Gateway = "ingressgateway"
	}

	prefix := "."
	if os.Getuid() == 0 {
		prefix = ""
	}
	for _, kv := range os.Environ() {
		kvl := strings.SplitN(kv, "=", 2)
		if strings.HasPrefix(kvl[0], "K8S_SECRET_") {
			kr.Secrets2Dirs[kvl[0][11:]] = prefix + kvl[1]
			InitSecret(k8sClient, ns, kvl[0][11:], prefix + kvl[1])
		}
		if kvl[0] == "SSH_CA" && initDebug != nil {
			kr.SSHCA = kvl[1]
			initDebug(kr)
		}
	}
	kr.Client = k8sClient
	RefreshTokens(kr, prefix)

	proxyConfig := os.Getenv("PROXY_CONFIG")
	if proxyConfig == "" {
		xdsAddr := os.Getenv("XDS_ADDR")
		if xdsAddr != "" {
			proxyConfig = fmt.Sprintf(`{"discoveryAddress": "%s"}`, xdsAddr)
		}
	}
	if proxyConfig != "" {
		kr.StartIstioAgent(proxyConfig)
	}

	if kr.Gateway == "" {
		startApp()
	}

	select{}
}


// Refresh the tokens
func RefreshTokens(kr *k8s.KRun, prefix string) {
	kr.Aud2File = map[string]string{}
	for _, kv := range os.Environ() {
		kvl := strings.SplitN(kv, "=", 2)
		if strings.HasPrefix(kvl[0], "K8S_TOKEN_") {
			kr.Aud2File[kvl[0][10:]] =  prefix + kvl[1]
		}
	}
	kr.Aud2File["istio-ca"] = prefix + "/var/run/secrets/tokens/istio-token"
	kr.Aud2File["api"] = prefix + "/var/run/secrets/kubernetes.io/serviceaccount/token"

	if kr.KSA == "" {
		kr.KSA = "default"
	}

	kr.Refresh()
}


// startApp uses the reminder of the command line to exec an app, using K8S_UID as UID, if present.
func startApp() {
	var cmd *exec.Cmd
	if len(os.Args) == 2 {
		cmd = exec.Command(os.Args[1])
	} else {
		cmd = exec.Command(os.Args[1], os.Args[2:]...)
	}
	if os.Getuid() == 0 {
		uid := os.Getenv("K8S_UID")
		if uid != "" {
			uidi, err := strconv.Atoi(uid)
			if err == nil {
				cmd.SysProcAttr = &syscall.SysProcAttr{}
				cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uidi)}
			}
		}
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go func() {
		err := cmd.Start()
		if err != nil {
			log.Println("Failed to start ", cmd, err)
		}
		err = cmd.Wait()
		if err != nil {
			log.Println("Failed to wait ", cmd, err)
		}
		os.Exit(0)
	}()
}


func InitSecret(k8sClient *kubernetes.Clientset,  ns string, name string, path string) {
	s, err := k8sClient.CoreV1().Secrets(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
			panic(err)
		}
	for k, v := range s.Data {
		err = ioutil.WriteFile(path + "/" + k, v, 0700)
		if err != nil {
			log.Println("Failed to init secret ", name, path, k, err)
		}
	}
}


