package mesh

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Audience to files. For each key, a k8s token with the given audience
	// will be created. Files should be under /var/run/secrets
	Aud2File map[string]string

	// ProxyConfig is a subset of istio ProxyConfig
	ProxyConfig *ProxyConfig

	// Address of the XDS server. If not specified, MCP is used.
	XDSAddr string

	// IstiodTenant, extracted from cluster. Only set if using MCP or external Istiod
	IstiodTenant string

	MeshConnectorAddr         string
	MeshConnectorInternalAddr string

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

	// MeshAddr is the location of the mesh environment file.
	MeshAddr    string
	CitadelRoot string
}

func New(addr string) *KRun {
	kr := &KRun{
		MeshAddr: addr,
		StartTime:    time.Now(),
		Aud2File:     map[string]string{},
		CM2Dirs:      map[string]string{},
		Labels:       map[string]string{},
		Secrets2Dirs: map[string]string{},
		ProxyConfig:  &ProxyConfig{},
	}
	return kr
}

// initFromEnv will use the env variables, metadata server and cluster configmaps
// to get the initial configuration for Istio and KRun.
//
func (kr *KRun) initFromEnv()  {

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
	if kr.IstiodTenant == "" {
		kr.IstiodTenant = os.Getenv("ISTIOD_TENANT")
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

	// TODO: stop using this, use ProxyConfig.DiscoveryAddress instead
	if kr.XDSAddr == "" {
		kr.XDSAddr = os.Getenv("XDS_ADDR")
	}

	pc := os.Getenv("PROXY_CONFIG")
	if pc != "" {
		err := yaml.Unmarshal([]byte(pc), &kr.ProxyConfig)
		if err != nil {
			log.Println("Invalid ProxyConfig, ignoring", err)
		}
		if kr.ProxyConfig.DiscoveryAddress != "" {
			kr.XDSAddr = kr.ProxyConfig.DiscoveryAddress
		}
	}

	// Advanced options

	// example dns:debug
	kr.AgentDebug = cfg("XDS_AGENT_DEBUG", "")
}

// RefreshAndSaveFiles is run periodically to create token, secrets, config map files.
// The primary use is istio token expected by pilot agent. This should not be called unless pilot-agent
// and envoy are used. Certs for proxyless gRPC and 'direct' use can be created without saving the tokens.
func (kr *KRun) RefreshAndSaveFiles() {
	for aud, f := range kr.Aud2File {
		kr.saveTokenToFile(kr.Namespace, aud, f)
	}
	for k, v := range kr.Secrets2Dirs {
		kr.saveSecretToFile(k, v)
	}
	for k, v := range kr.CM2Dirs {
		kr.saveConfigMapToFile(k, v)
	}

	time.AfterFunc(30*time.Minute, kr.RefreshAndSaveFiles)
}

// Internal implementation detail for the 'mesh-env' for Istio and MCP.
// This may change, it is not a stable API - see loadMeshEnv for the other side.
func (kr *KRun) SaveToMap(d map[string]string) bool {
	needUpdate := false

	// Set the GCP specific options, extracted from metadata - if not already set.
	needUpdate = setIfEmpty(d, "PROJECT_NUMBER", kr.ProjectNumber, needUpdate)
	needUpdate = setIfEmpty(d, "ISTIOD_TENANT", kr.IstiodTenant, needUpdate)
	needUpdate = setIfEmpty(d, "XDS_ADDR", kr.XDSAddr, needUpdate)
	needUpdate = setIfEmpty(d, "CLUSTER_NAME", kr.ClusterName, needUpdate)
	needUpdate = setIfEmpty(d, "CLUSTER_LOCATION", kr.ClusterLocation, needUpdate)
	needUpdate = setIfEmpty(d, "PROJECT_ID", kr.ProjectId, needUpdate)
	needUpdate = setIfEmpty(d, "MCON_ADDR", kr.MeshConnectorAddr, needUpdate)
	needUpdate = setIfEmpty(d, "IMCON_ADDR", kr.MeshConnectorInternalAddr, needUpdate)
	needUpdate = setIfEmpty(d, "ISTIOD_ROOT", kr.CitadelRoot, needUpdate)

	return needUpdate
}

// loadMeshEnv will lookup the 'mesh-env', an opaque config for the mesh.
// Currently it is loaded from K8S
// TODO: URL, like 'konfig' ( including gcp pseudo-URL like gcp://cluster.location.project/.... )
//
func (kr *KRun) loadMeshEnv(ctx context.Context) error {
	s, err := kr.Client.CoreV1().ConfigMaps("istio-system").Get(ctx,
		"mesh-env", metav1.GetOptions{})
	if err != nil {
		if Is404(err) {
			return nil
		}
		return err
	}
	d := s.Data
	// See connector for supported values
	updateFromMap(d, "PROJECT_NUMBER", &kr.ProjectNumber)
	updateFromMap(d, "ISTIOD_TENANT", &kr.IstiodTenant)
	updateFromMap(d, "XDS_ADDR", &kr.XDSAddr)
	updateFromMap(d, "CLUSTER_NAME", &kr.ClusterName)
	updateFromMap(d, "CLUSTER_LOCATION", &kr.ClusterLocation)
	updateFromMap(d, "PROJECT_ID", &kr.ProjectId)
	updateFromMap(d, "MCON_ADDR", &kr.MeshConnectorAddr)
	updateFromMap(d, "IMCON_ADDR", &kr.MeshConnectorInternalAddr)
	updateFromMap(d, "ISTIOD_ROOT", &kr.CitadelRoot)

	// Old style:
	updateFromMap(d, "CLOUDRUN_ADDR", &kr.IstiodTenant)

	return nil
}

func setIfEmpty(d map[string]string, key, val string, upd bool) bool {
	if d[key] == "" && val != "" {
		d[key] = val
		return true
	}
	return upd
}

func updateFromMap(d map[string]string, key string, dest *string) {
	if d[key] != "" && *dest == "" {
		*dest = d[key]
	}
}

func cfg(name, def string) string {
	v := os.Getenv(name)
	if name == "" {
		return def
	}
	return v
}

func Is404(err error) bool {
	if se, ok := err.(*errors.StatusError); ok {
		if se.ErrStatus.Code == 404 {
			return true
		}
	}
	return false
}
