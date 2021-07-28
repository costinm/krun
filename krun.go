package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/costinm/krun/pkg/hbone"
	"github.com/costinm/krun/pkg/k8s"
)

var initDebug func(run *k8s.KRun)

func cfg(name, def string) string {
	v := os.Getenv(name)
	if name == "" {
		return def
	}
	return v
}

func main() {
	kr := &k8s.KRun{}

	k8sClient, err := kr.GetK8S()
	if err != nil {
		panic(err)
	}

	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		ns = os.Getenv("K8S_NS")
	}
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
	kr.Name = name
	kr.Namespace = ns

	// example dns:debug
	kr.AgentDebug = cfg("XDS_AGENT_DEBUG", "")

	if len(os.Args) == 1 {
		// Default gateway label for now, we can customize with env variables.
		kr.Gateway = "ingressgateway"
		log.Println("Starting in gateway mode", os.Args)
	}

	prefix := "."
	kr.Client = k8sClient
	kr.Aud2File = map[string]string{}
	if os.Getuid() == 0 {
		prefix = ""
	}
	for _, kv := range os.Environ() {
		kvl := strings.SplitN(kv, "=", 2)
		if strings.HasPrefix(kvl[0], "K8S_SECRET_") {
			kr.Secrets2Dirs[kvl[0][11:]] = prefix + kvl[1]
		}
		if strings.HasPrefix(kvl[0], "K8S_CM_") {
			kr.CM2Dirs[kvl[0][7:]] = prefix + kvl[1]
		}
		if kvl[0] == "SSH_CA" && initDebug != nil {
			kr.SSHCA = kvl[1]
			// Split for conditional compilation (to compile without ssh dep)
			initDebug(kr)
		}
		if strings.HasPrefix(kvl[0], "K8S_TOKEN_") {
			kr.Aud2File[kvl[0][10:]] =  prefix + kvl[1]
		}
	}
	aud := os.Getenv("TRUST_DOMAIN")
	if aud == "" {
		aud = os.Getenv("PROJECT") + ".svc.id.goog"
	}
	kr.Aud2File[aud] = prefix + "/var/run/secrets/tokens/istio-token"
	kr.Aud2File["api"] = prefix + "/var/run/secrets/kubernetes.io/serviceaccount/token"

	if kr.KSA == "" {
		kr.KSA = "default"
	}

	kr.Refresh()
	xdsAddr := os.Getenv("XDS_ADDR")
	log.Println("============== Starting with ", xdsAddr, aud)
	if xdsAddr != "" {
		proxyConfig := fmt.Sprintf(`{"discoveryAddress": "%s"}`, xdsAddr)
		kr.StartIstioAgent(proxyConfig)
	}

	kr.StartApp()

	// TODO: wait for app and proxy ready
	hb := &hbone.HBone{

	}
	err = hb.Init()
	if err != nil {
		panic(err)
	}

	err = hb.Start(":14009")
	if err != nil {
		panic(err)
	}

	select{}
}
