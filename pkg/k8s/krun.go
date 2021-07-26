package k8s

import (
	"os/exec"

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

	// URL of a SSH Cert authority, similar with Istiod.
	// If set, will be used to enable an SSHD server, with a cert signed
	// by the CA based on the Istio mTLS certificate, with the same identity.
	//
	// The SSH server will accept connections using certs signed by the same
	// cert authority, with same namespace or istio-system.
	SSHCA string

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
	Client   *kubernetes.Clientset

	// List of clusters - used if location and cluster are not set explicitly
	clusters []*containerpb.Cluster

	// Kubeconfig - constructed by looking up the clusters
	KubeConfig *clientcmdapi.Config

	agentCmd *exec.Cmd
	appCmd   *exec.Cmd
}

type cluster struct {
	client   *kubernetes.Clientset

}
