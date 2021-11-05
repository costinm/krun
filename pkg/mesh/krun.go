// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mesh

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Cfg interface {
	GetSecret(ctx context.Context, ns string, name string) (map[string][]byte, error)
	GetCM(ctx context.Context, ns string, name string) (map[string]string, error)
}

type TokenProvider interface {
	GetToken(ctx context.Context, aud string) (string, error)
}

// KRun allows running an app in an Istio and K8S environment.
type KRun struct {
	// BaseDir is the root directory for all created files and all lookups.
	// If empty, will default to "/" when running as root, and "./" when running as regular user.
	// MESH_BASE_DIR will override it.
	BaseDir string

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

	// MeshTenant. Only set if using MCP or external Istiod.
	// Opaque, internal string that identifies the mesh to the XDS server.
	// Different from meshID - which is the user-visible form.
	MeshTenant string

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

	ProjectId     string
	ProjectNumber string
	ClusterName   string
	// Location of the cluster
	ClusterLocation string

	agentCmd    *exec.Cmd
	appCmd      *exec.Cmd
	TrustDomain string

	StartTime time.Time
	Labels    map[string]string

	// WhiteboxMode indicates no iptables capture
	WhiteboxMode bool
	InCluster    bool

	// PEM cert roots detected in the cluster - Citadel, custom CAs from mesh config.
	// Will be saved to a file.
	CARoots []string

	// MeshAddr is the location of the mesh environment file.
	MeshAddr    string
	CitadelRoot string
	InstanceID  string

	// Interface to abstract k8s implementation
	TokenProvider    TokenProvider
	Cfg              Cfg
	TransportWrapper func(transport http.RoundTripper) http.RoundTripper
}

var Debug = false

func New(addr string) *KRun {
	kr := &KRun{
		MeshAddr:    addr,
		StartTime:   time.Now(),
		Aud2File:    map[string]string{},
		Labels:      map[string]string{},
		ProxyConfig: &ProxyConfig{},
	}
	kr.initFromEnv()
	return kr
}

// Extract Region from ClusterLocation
func (kr *KRun) Region() string {
	p := strings.Split(kr.ClusterLocation, "-")
	if len(p) < 3 {
		return kr.ClusterLocation
	}
	return strings.Join(p[0:2], "-")
}

// initFromEnv will use the env variables, metadata server and cluster configmaps
// to get the initial configuration for Istio and KRun.
//
func (kr *KRun) initFromEnv() {

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
	if kr.MeshTenant == "" {
		kr.MeshTenant = os.Getenv("MESH_TENANT")
	}

	ks := os.Getenv("K_SERVICE")
	if kr.Name == "" {
		verNsName := strings.SplitN(ks, "--", 2)
		if len(verNsName) > 1 {
			ks = verNsName[1]
			kr.Labels["ver"] = verNsName[0]
		} else {
			kr.Name = ks
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
	} else {
		kr.BaseDir = prefix
	}

	if kr.TrustDomain == "" {
		kr.TrustDomain = os.Getenv("TRUST_DOMAIN")
	}
	if kr.TrustDomain == "" && kr.ProjectId != "" {
		kr.TrustDomain = kr.ProjectId + ".svc.id.goog"
	}
	// This can be used to provide a k8s-like environment, for apps that need it.
	// It might be better to just generate a kubeconfig file and not pretend we are inside a cluster.
	//if !kr.InCluster {
	//	kr.Aud2File["api"] = prefix + "/var/run/secrets/kubernetes.io/serviceaccount/token"
	//}
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

// RefreshAndSaveTokens is run periodically to create token, secrets, config map files.
// The primary use is istio token expected by pilot agent.
// This should not be called unless pilot-agent/envoy  or proxyless gRPC without library are used.
// pilot-agent is currently refreshing the certificates - WIP to move that here.
//
// Certs for 'direct' (library) use can be created without saving the tokens.
// 'library' means linking this or a similar package with the application.
func (kr *KRun) RefreshAndSaveTokens() {
	for aud, f := range kr.Aud2File {
		kr.saveTokenToFile(kr.Namespace, aud, f)
	}
	time.AfterFunc(30*time.Minute, kr.RefreshAndSaveTokens)
}

func (kr *KRun) saveTokenToFile(ns string, audience string, destFile string) error {
	t, err := kr.GetToken(context.TODO(), audience)
	if err != nil {
		log.Println("Error creating ", ns, kr.KSA, audience, err)
		return err
	}
	lastSlash := strings.LastIndex(destFile, "/")
	err = os.MkdirAll(destFile[:lastSlash], 0755)
	if err != nil {
		log.Println("Error creating dir", ns, kr.KSA, destFile[:lastSlash])
	}
	// Save the token, readable by app. Little value to have istio token as different user,
	// for this separate container/sandbox is needed.
	err = ioutil.WriteFile(destFile, []byte(t), 0644)
	if err != nil {
		log.Println("Error creating ", ns, kr.KSA, audience, destFile, err)
		return err
	}

	return nil
}

// Internal implementation detail for the 'mesh-env' for Istio and MCP.
// This may change, it is not a stable API - see loadMeshEnv for the other side.
func (kr *KRun) SaveToMap(d map[string]string) bool {
	needUpdate := false

	// Set the GCP specific options, extracted from metadata - if not already set.
	needUpdate = setIfEmpty(d, "PROJECT_NUMBER", kr.ProjectNumber, needUpdate)
	needUpdate = setIfEmpty(d, "MESH_TENANT", kr.MeshTenant, needUpdate)
	needUpdate = setIfEmpty(d, "XDS_ADDR", kr.XDSAddr, needUpdate)
	needUpdate = setIfEmpty(d, "CLUSTER_NAME", kr.ClusterName, needUpdate)
	needUpdate = setIfEmpty(d, "CLUSTER_LOCATION", kr.ClusterLocation, needUpdate)
	needUpdate = setIfEmpty(d, "PROJECT_ID", kr.ProjectId, needUpdate)
	needUpdate = setIfEmpty(d, "MCON_ADDR", kr.MeshConnectorAddr, needUpdate)
	needUpdate = setIfEmpty(d, "IMCON_ADDR", kr.MeshConnectorInternalAddr, needUpdate)

	// TODO: use CAROOT_XXX to save multiple CAs (MeshCA, Citadel, other clusters)
	needUpdate = setIfEmpty(d, "CAROOT_ISTIOD", kr.CitadelRoot, needUpdate)

	return needUpdate
}

// loadMeshEnv will lookup the 'mesh-env', an opaque config for the mesh.
// Currently it is loaded from K8S
// TODO: URL, like 'konfig' ( including gcp pseudo-URL like gcp://cluster.location.project/.... )
//
func (kr *KRun) loadMeshEnv(ctx context.Context) error {
	d, err := kr.Cfg.GetCM(ctx, "istio-system", "mesh-env")
	if err != nil {
		return err
	}
	return kr.initFromMap(d)
}

func (kr *KRun) initFromMap(d map[string]string) error {
	// See connector for supported values
	updateFromMap(d, "PROJECT_NUMBER", &kr.ProjectNumber)
	updateFromMap(d, "MESH_TENANT", &kr.MeshTenant)
	updateFromMap(d, "XDS_ADDR", &kr.XDSAddr)
	updateFromMap(d, "CLUSTER_NAME", &kr.ClusterName)
	updateFromMap(d, "CLUSTER_LOCATION", &kr.ClusterLocation)
	updateFromMap(d, "PROJECT_ID", &kr.ProjectId)
	updateFromMap(d, "MCON_ADDR", &kr.MeshConnectorAddr)
	updateFromMap(d, "IMCON_ADDR", &kr.MeshConnectorInternalAddr)
	updateFromMap(d, "CAROOT_ISTIOD", &kr.CitadelRoot)

	if kr.CitadelRoot != "" {
		kr.CARoots = append(kr.CARoots, kr.CitadelRoot)
	}
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
