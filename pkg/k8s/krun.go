package k8s

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
)

// KRun allows running an app in an Istio and K8S environment.
type KRun struct {
	// BaseDir is the root directory for all created files and all lookups.
	// If empty, will default to "/" when running as root, and "./" when running as regular user.
	// MESH_BASE_DIR will override it.
	BaseDir string

	// Secrets to 'mount'. Key is the secret name, value is a path.
	// All secret mounts are 'optional=true' ( for now )
	Secrets2Dirs map[string]string

	// Config maps to 'mount'. Key is the config map name, value is a path.
	// Config mounts are optional (for now)
	CM2Dirs map[string]string

	ExtraEnv []string

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

	ProjectId       string
	ProjectNumber   string
	ClusterName     string
	ClusterLocation string

	agentCmd    *exec.Cmd
	appCmd      *exec.Cmd
	TrustDomain string

	StartTime  time.Time
	Labels     map[string]string
	VendorInit func(context.Context, *KRun) error

	// WhiteboxMode indicates no iptables capture
	WhiteboxMode bool
	InCluster    bool

	// PEM cert roots detected in the cluster - Citadel, custom CAs from mesh config.
	// Will be saved to a file.
	CARoots []string
}

func New() *KRun {
	kr := &KRun{
		StartTime:    time.Now(),
		Aud2File:     map[string]string{},
		CM2Dirs:      map[string]string{},
		Labels:       map[string]string{},
		Secrets2Dirs: map[string]string{},
	}
	return kr
}

// LoadConfig will use the env variables, metadata server and cluster configmaps
// to get the initial configuration for Istio and KRun.
//
//
func (kr *KRun) LoadConfig() *KRun {

	if kr.KSA == "" {
		// Same environment used for VMs
		kr.KSA = os.Getenv("WORKLOAD_SERVICE_ACCOUNT")
	}
	if kr.KSA == "" {
		kr.KSA = "default"
	}

	if kr.Namespace == "" {
		// Same environment used for VMs
		kr.Namespace = os.Getenv("WORKLOAD_NAMESPACE")
	}
	if kr.Name == "" {
		kr.Name = os.Getenv("WORKLOAD_NAME")
	}
	if kr.Gateway == "" {
		kr.Gateway = os.Getenv("GATEWAY_NAME")
	}

	ks := os.Getenv("K_SERVICE")
	if kr.Namespace == "" {
		verNsName := strings.SplitN(ks, "--", 2)
		if len(verNsName) > 1 {
			ks = verNsName[1]
			kr.Labels["ver"] = verNsName[0]
		}
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
	if kr.BaseDir == "" {
		kr.BaseDir = os.Getenv("MESH_BASE_DIR")
	}
	if kr.BaseDir != "" {
		prefix = kr.BaseDir
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
			kr.Aud2File[kvl[0][10:]] = prefix + kvl[1]
		}
		if strings.HasPrefix(kvl[0], "LABEL_") {
			kr.Labels[kvl[0][6:]] = prefix + kvl[1]
		}
	}

	if kr.TrustDomain == "" {
		kr.TrustDomain = os.Getenv("TRUST_DOMAIN")
	}
	if kr.TrustDomain == "" {
		kr.TrustDomain = kr.ProjectId + ".svc.id.goog"
	}
	kr.Aud2File[kr.TrustDomain] = prefix + "/var/run/secrets/tokens/istio-token"
	if !kr.InCluster {
		kr.Aud2File["api"] = prefix + "/var/run/secrets/kubernetes.io/serviceaccount/token"
	}
	if kr.KSA == "" {
		kr.KSA = "default"
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
