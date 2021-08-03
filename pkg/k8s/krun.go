package k8s

import (
	"os"
	"os/exec"
	"strings"

	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// KRun allows running an app in an Istio and K8S environment.
type KRun struct {
	// Secrets to 'mount'. Key is the secret name, value is a path.
	// All secret mounts are 'optional=true' ( for now )
	Secrets2Dirs map[string]string

	// Config maps to 'mount'. Key is the config map name, value is a path.
	// Config mounts are optional (for now)
	CM2Dirs map[string]string

	// Audience to files. For each key, a k8s token with the given audience
	// will be created. Files should be under /var/run/secrets
	Aud2File map[string]string

	// Address of the XDS server. If not specified, MCP is used.
	XDSAddr string
	// MCPAddr, extracted from cluster. Only set if using MCP
	MCPAddr string

	// Canonical name for the application.
	// Will be set as "app" and "service.istio.io/canonical-name" labels
	//
	// If not set "default" will be used.
	// TODO: use service name as default
	Name string

	// If not empty, will run Istio-agent as a gateway (router instead of sidecar)
	// with the "istio: $Gateway" label.
	Gateway string

	// Agent debug config (example dns:debug).
	// Based on ISTIO_DEBUG
	AgentDebug string

	// Namespace for the application. The user running the command must have
	// the appropriate Token, Secret, ConfigMap permissions in the namespace.
	//
	// If not set, "default" will be used.
	// TODO: use the GSA name as default namespace.
	Namespace string

	// KSA is the k8s service account for getting tokens.
	//
	// If not set, "default" will be used.
	// TODO: use service name as default
	KSA string

	// Primary client is the k8s client to use. If not set will be created based on
	// the config.
	Client *kubernetes.Clientset

	// List of Clusters - used if location and cluster are not set explicitly
	Clusters []*containerpb.Cluster

	// Kubeconfig - constructed by looking up the clusters
	KubeConfig *clientcmdapi.Config

	ProjectId       string
	ProjectNumber   string
	ClusterName     string
	ClusterLocation string

	agentCmd    *exec.Cmd
	appCmd      *exec.Cmd
	TrustDomain string
}

func (kr *KRun) InitFromEnv() *KRun {

	if kr.KSA == "" {
		kr.KSA = os.Getenv("WORKLOAD_SERVICE_ACCOUNT")
	}

	if kr.KSA == "" {
		kr.KSA = "default"
	}

	if kr.Namespace == "" {
		kr.Namespace = os.Getenv("WORKLOAD_NAMESPACE")
	}
	if kr.Name == "" {
		kr.Name = os.Getenv("WORKLOAD_NAME")
	}

	ks := os.Getenv("K_SERVICE")
	if kr.Namespace == "" {
		// TODO: revision--NS-NAME-SUFFIX
		parts := strings.Split(ks, "-")
		kr.Namespace = parts[0]
		if len(parts) > 1 {
			kr.Name = parts[1]
		}
	}

	if kr.Namespace == "" {
		kr.Namespace = "default"
	}
	if kr.Name == "" {
		kr.Name = kr.Namespace
	}

	kr.Aud2File = map[string]string{}
	prefix := "."
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
		if strings.HasPrefix(kvl[0], "K8S_TOKEN_") {
			kr.Aud2File[kvl[0][10:]] =  prefix + kvl[1]
		}
	}
	if kr.ProjectId == "" {
		kr.ProjectId = os.Getenv("PROJECT_ID")
	}
	if kr.ProjectId == "" {
		kr.ProjectId, _ = ProjectFromMetadata()
	}

	if kr.TrustDomain == "" {
		kr.TrustDomain = os.Getenv("TRUST_DOMAIN")
	}
	if kr.TrustDomain == "" {
		kr.TrustDomain = kr.ProjectId + ".svc.id.goog"
	}
	kr.Aud2File[kr.TrustDomain] = prefix + "/var/run/secrets/tokens/istio-token"
	kr.Aud2File["api"] = prefix + "/var/run/secrets/kubernetes.io/serviceaccount/token"

	if kr.KSA == "" {
		kr.KSA = "default"
	}

	if kr.ClusterName == "" {
		kr.ClusterName = os.Getenv("CLUSTER_NAME")
	}


	if kr.ProjectNumber == "" {
		kr.ProjectNumber = os.Getenv("PROJECT_NUMBER")
	}
	if kr.ProjectNumber == "" {
		kr.ProjectNumber, _ = ProjectNumberFromMetadata()
	}


	if kr.ClusterLocation == "" {
		kr.ClusterLocation = os.Getenv("CLUSTER_LOCATION")
	}
	// Deprecated
	if kr.ClusterLocation == "" {
		kr.ClusterLocation = os.Getenv("LOCATION")
	}
	if kr.ClusterLocation == "" {
		kr.ClusterLocation, _ = RegionFromMetadata()
	}

	if kr.XDSAddr == "" {
		kr.XDSAddr = os.Getenv("XDS_ADDR")
	}
	// Advanced options

	// example dns:debug
	kr.AgentDebug = cfg("XDS_AGENT_DEBUG", "")

	return kr
}

func cfg(name, def string) string {
	v := os.Getenv(name)
	if name == "" {
		return def
	}
	return v
}
