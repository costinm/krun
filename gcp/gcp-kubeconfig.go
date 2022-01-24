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

package gcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	container "cloud.google.com/go/container/apiv1"
	"github.com/costinm/krun/k8s/k8s"

	"github.com/costinm/krun/pkg/mesh"

	"k8s.io/client-go/kubernetes"
	kubeconfig "k8s.io/client-go/tools/clientcmd/api"
	// Required for k8s client to link in the authenticator
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	gkehub "cloud.google.com/go/gkehub/apiv1beta1"
	crm "google.golang.org/api/cloudresourcemanager/v1"

	gkehubpb "google.golang.org/genproto/googleapis/cloud/gkehub/v1beta1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

// Integration with GCP - use metadata server or GCP-specific env variables to auto-configure connection to a
// GKE cluster and extract metadata.

// Using the metadata package, which connects to 169.254.169.254, metadata.google.internal or $GCE_METADATA_HOST (http, no prefix)
// Will attempt to guess if running on GCP if env variable is not set.
// Note that requests are using a 2 sec timeout.

// TODO:  finish hub.

// Cluster wraps cluster information for a discovered hub or gke cluster.
type Cluster struct {
	ClusterName     string
	ClusterLocation string
	ProjectId       string

	GKECluster *containerpb.Cluster
	HubCluster *gkehubpb.Membership

	KubeConfig *kubeconfig.Config
}

var (
	GCPInitTime time.Duration
)

// configFromEnvAndMD will attempt to configure ProjectId, ClusterName, ClusterLocation, ProjectNumber, used on GCP
// Metadata server will be tried if env variables don't exist.
func configFromEnvAndMD(ctx context.Context, kr *k8s.K8S) {
	if kr.Mesh.ProjectId == "" {
		kr.Mesh.ProjectId = os.Getenv("PROJECT_ID")
	}

	if kr.Mesh.ClusterName == "" {
		kr.Mesh.ClusterName = os.Getenv("CLUSTER_NAME")
	}

	if kr.Mesh.ClusterLocation == "" {
		kr.Mesh.ClusterLocation = os.Getenv("CLUSTER_LOCATION")
	}

	if kr.Mesh.ProjectNumber == "" {
		kr.Mesh.ProjectNumber = os.Getenv("PROJECT_NUMBER")
	}

	// If ADC is set, we will only use the env variables. Else attempt to init from metadata server.
	if os.Getenv("APPLICATION_DEFAULT_CREDENTIALS") != "" {
		return
	}

	t0 := time.Now()
	if metadata.OnGCE() {
		// TODO: detect if the cluster is k8s from some env ?
		// If ADC is set, we will only use the env variables. Else attempt to init from metadata server.
		metaProjectId, _ := metadata.ProjectID()
		if kr.Mesh.ProjectId == "" {
			kr.Mesh.ProjectId = metaProjectId
		}

		//instanceID, _ := metadata.InstanceID()
		//instanceName, _ := metadata.InstanceName()
		//zone, _ := metadata.Zone()
		//pAttr, _ := metadata.ProjectAttributes()
		//hn, _ := metadata.Hostname()
		//iAttr, _ := metadata.InstanceAttributes()

		email, _ := metadata.Email("default")
		// In CloudRun: Additional metadata: iid 00bf4b...f23 iname  iattr [] zone us-central1-1 hostname  pAttr [] email k8s-fortio@wlhe-cr.iam.gserviceaccount.com

		//log.Println("Additional metadata:", "iid", instanceID, "iname", instanceName, "iattr", iAttr,
		//	"zone", zone, "hostname", hn, "pAttr", pAttr, "email", email)

		if kr.Mesh.Namespace == "" {
			if strings.HasPrefix(email, "k8s-") {
				parts := strings.Split(email[4:], "@")
				kr.Mesh.Namespace = parts[0]
				if mesh.Debug {
					log.Println("Defaulting Namespace based on email: ", kr.Mesh.Namespace, email)
				}
			}
		}
		var err error
		if kr.Mesh.InCluster && kr.Mesh.ClusterName == "" {
			kr.Mesh.ClusterName, err = metadata.Get("instance/attributes/cluster-name")
			if err != nil {
				log.Println("Can't find cluster name")
			}
		}
		if kr.Mesh.InCluster && kr.Mesh.ClusterLocation == "" {
			kr.Mesh.ClusterLocation, err = metadata.Get("instance/attributes/cluster-location")
			if err != nil {
				log.Println("Can't find cluster location")
			}
		}

		if kr.Mesh.InstanceID == "" {
			kr.Mesh.InstanceID, _ = metadata.InstanceID()
		}
		if mesh.Debug {
			log.Println("Configs from metadata ", time.Since(t0))
		}
		log.Println("Running as GSA ", email, kr.Mesh.ProjectId, kr.Mesh.ProjectNumber, kr.Mesh.InstanceID, kr.Mesh.ClusterLocation)
	}
}

func RegionFromMetadata() (string, error) {
	v, err := metadata.Get("instance/region")
	if err != nil {
		return "", err
	}
	vs := strings.SplitAfter(v, "/regions/")
	if len(vs) != 2 {
		return "", fmt.Errorf("malformed region value split into %#v", vs)
	}
	return vs[1], nil
}

func TokenGKE(ctx context.Context, aud string) (string, error) {
	uri := fmt.Sprintf("instance/service-accounts/default/identity?audience=%s", aud)
	tok, err := metadata.Get(uri)
	if err != nil {
		return "", err
	}
	return tok, nil
}

func Token(ctx context.Context, aud string) (string, error) {
	uri := fmt.Sprintf("instance/service-accounts/default/identity?audience=%s&format=full", aud)
	tok, err := metadata.Get(uri)
	if err != nil {
		return "", err
	}
	return tok, nil
}

func ConfigGCP(ctx context.Context, kr *mesh.KRun) error {
	// Avoid direct dependency on GCP libraries - may be replaced by a REST client or different XDS server discovery.
	kc := &k8s.K8S{Mesh: kr, VendorInit: gcp.InitGCP}
	err := kc.K8SClient(ctx)

	return err
}

// InitGCP loads GCP-specific metadata and discovers the config cluster.
// This step is skipped if user has explicit configuration for required settings.
//
// Namespace,
// ProjectId, ProjectNumber
// ClusterName, ClusterLocation
func InitGCP(ctx context.Context, kr *k8s.K8S) error {
	// Load GCP env variables - will be needed.
	configFromEnvAndMD(ctx, kr)

	if kr.Client != nil {
		// Running in-cluster or using kube config
		return nil
	}

	t0 := time.Now()
	var kc *kubeconfig.Config
	var err error

	// TODO: attempt to get the config project ID from a label on the workload or project
	// (if metadata servere or CR can provide them)

	configProjectID := kr.Mesh.ProjectId
	configLocation := kr.Mesh.ClusterLocation
	configClusterName := krMesh..ClusterName

	if kr.Mesh.MeshAddr != nil {
		if kr.Mesh.MeshAddr.Scheme == "gke" {
			configProjectID = kr.MeshAddr.Host
		} else if kr.MeshAddr.Host == "container.googleapis.com" {
			// Not using the hub resourceLink format:
			//    container.googleapis.com/projects/wlhe-cr/locations/us-central1-c/clusters/asm-cr
			// or the 'selfLink' from GKE list API
			// "https://container.googleapis.com/v1/projects/wlhe-cr/locations/us-west1/clusters/istio"

			configProjectID = kr.Mesh.MeshAddr.Host
			if len(kr.Mesh.MeshAddr.Path) > 1 {
				parts := strings.Split(kr.Mesh.MeshAddr.Path, "/")
				for i := 0 ; i < len(parts); i++ {
					if parts[i] == "projects" && i+1 < len(parts) {
						configProjectID = parts[i+1]
					}
					if parts[i] == "locations" && i+1 < len(parts) {
						configLocation = parts[i+1]
					}
					if parts[i] == "clusters" && i+1 < len(parts) {
						configClusterName = parts[i+1]
					}
				}
			}
		}
	}

	if configProjectID == "" {
		// GCP can't be initialized without a project ID
		return nil
	}

	var cl *Cluster
	if configLocation == "" || configClusterName == "" {
		// ~500ms
		label := "mesh_id"
		// Try to get the region from metadata server. For Cloudrun, this is not the same with the cluster - it may be zonal
		myRegion, _ := RegionFromMetadata()
		if myRegion == "" {
			myRegion = configLocation
		}
		if kr.MeshAddr != nil && kr.MeshAddr.Scheme == "gke" {
			// Explicit mesh config clusters, no label selector ( used for ASM clusters in current project )
			label = ""
		}
		log.Println("Selecting a GKE cluster ", kr.ProjectId, configProjectID, myRegion)
		cll, err := AllClusters(ctx, kr, configProjectID, label, "")
		if err != nil {
			return err
		}

		if len(cll) == 0 {
			return nil // no cluster to use
		}


		cl = findCluster(kr, cll, myRegion, cl)
		// TODO: connect to cluster, find istiod - and keep trying until a working one is found ( fallback )
	} else {
		// Explicit override - user specified the full path to the cluster.
		// ~400 ms
		cl, err = GKECluster(ctx, configProjectID, kr.Mesh.ClusterLocation, kr.Mesh.ClusterName)
		if err != nil {
			return err
		}
		if err != nil {
			log.Println("Failed in NewForConfig", kr, err)
			return err
		}
	}

	kr.ProjectId = configProjectID

	kr.TrustDomain = configProjectID + ".svc.id.goog"
	kc = cl.KubeConfig
	if kr.Mesh.ClusterName == "" {
		kr.Mesh.ClusterName = cl.ClusterName
	}
	if kr.Mesh.ClusterLocation == "" {
		kr.Mesh.ClusterLocation = cl.ClusterLocation
	}
	kr.ClusterAddress =  fmt.Sprintf("https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s",
		configProjectID, kr.Mesh.ClusterLocation, kr.Mesh.ClusterName)

	GCPInitTime = time.Since(t0)

	rc, err := restConfig(kc)
	if err != nil {
		return err
	}
	kr.Client, err = kubernetes.NewForConfig(rc)
	if err != nil {
		return err
	}

	return nil
}

func restConfig(kc *kubeconfig.Config) (*rest.Config, error) {
	// TODO: set default if not set ?
	return clientcmd.NewNonInteractiveClientConfig(*kc, "", &clientcmd.ConfigOverrides{}, nil).ClientConfig()
}

func findCluster(kr *k8s.K8S, cll []*Cluster, myRegion string, cl *Cluster) *Cluster {
	if kr.Mesh.ClusterName != "" {
		for _, c := range cll {
			if myRegion != "" && !strings.HasPrefix(c.ClusterLocation, myRegion) {
				continue
			}
			if c.ClusterName == kr.Mesh.ClusterName {
				cl = c
				break
			}
		}
		if cl == nil {
			for _, c := range cll {
				if c.ClusterName == kr.Mesh.ClusterName {
					cl = c
					break
				}
			}
		}
	}

	// First attempt to find a cluster in same region, with the name prefix istio (TODO: label or other way to identify
	// preferred config clusters)
	if cl == nil {
		for _, c := range cll {
			if myRegion != "" && !strings.HasPrefix(c.ClusterLocation, myRegion) {
				continue
			}
			if strings.HasPrefix(c.ClusterName, "istio") {
				cl = c
				break
			}
		}
	}
	if cl == nil {
		for _, c := range cll {
			if myRegion != "" && !strings.HasPrefix(c.ClusterLocation, myRegion) {
				continue
			}
			cl = c
			break
		}
	}
	if cl == nil {
		for _, c := range cll {
			if strings.HasPrefix(c.ClusterName, "istio") {
				cl = c
			}
		}
	}
	// Nothing in same region, pick the first
	if cl == nil {
		cl = cll[0]
	}
	return cl
}

func GKECluster(ctx context.Context, p, l, clusterName string) (*Cluster, error) {
	cl, err := container.NewClusterManagerClient(ctx, option.WithQuotaProject(p))
	if err != nil {
		log.Println("Failed NewClusterManagerClient", p, l, clusterName, err)
		return nil, err
	}

	for i := 0; i < 5; i++ {
		gcr := &containerpb.GetClusterRequest{
			Name: fmt.Sprintf("projects/%s/locations/%s/cluster/%s", p, l, clusterName),
		}
		c, e := cl.GetCluster(ctx, gcr)
		if e == nil {
			rc := &Cluster{
				ProjectId:       p,
				ClusterLocation: c.Location,
				ClusterName:     c.Name,
				GKECluster:      c,
				KubeConfig:      addClusterConfig(c, p, l, clusterName),
			}

			return rc, nil
		}
		log.Println("Failed GetCluster, retry", gcr, p, l, clusterName, err)
		time.Sleep(1 * time.Second)
		err = e
	}
	return nil, err
}

func ProjectNumber(p string) string {
	ctx := context.Background()

	cr, err := crm.NewService(ctx)
	if err != nil {
		return ""
	}
	pdata, err := cr.Projects.Get(p).Do()
	if err != nil {
		log.Println("Error getting project number", p, err)
		return ""
	}

	// This is in v1 - v3 has it encoded in name.
	return strconv.Itoa(int(pdata.ProjectNumber))
}

// AllHub connects to GKE Hub and gets all clusters registered in the hub.
// TODO: document/validate GKE Connect auth mode
//
func AllHub(ctx context.Context, kr *k8s.K8S) ([]*Cluster, error) {
	cl, err := gkehub.NewGkeHubMembershipClient(ctx)
	if err != nil {
		return nil, err
	}

	mi := cl.ListMemberships(ctx, &gkehubpb.ListMembershipsRequest{
		Parent: "projects/" + kr.Mesh.ProjectId + "/locations/-",
	})

	// Also includes:
	// - labels
	// - Endpoint - including GkeCluster resource link ( the GKE name)
	// - State - should be READY
	//
	ml := []*Cluster{}
	for {
		r, err := mi.Next()
		//fmt.Println(r, err)
		if err != nil || r == nil {
			log.Println("Listing hub", kr.Mesh.ProjectId, err)
			break
		}
		mna := strings.Split(r.Name, "/")
		mn := mna[len(mna)-1]
		ctxName := "connectgateway_" + kr.Mesh.ProjectId + "_" + mn
		kc := kubeconfig.NewConfig()
		kc.Contexts[ctxName] = &kubeconfig.Context{
			Cluster:  ctxName,
			AuthInfo: ctxName,
		}
		kc.Clusters[ctxName] = &kubeconfig.Cluster{
			Server: fmt.Sprintf("https://connectgateway.googleapis.com/v1beta1/projects/%s/memberships/%s",
				kr.Mesh.ProjectNumber, mn),
		}
		kc.AuthInfos[ctxName] = &kubeconfig.AuthInfo{
			AuthProvider: &kubeconfig.AuthProviderConfig{
				Name: "gcp",
			},
		}

		// TODO: better way to select default
		kc.CurrentContext = ctxName

		c := &Cluster{
			ProjectId:   kr.Mesh.ProjectId,
			ClusterName: r.Name,
			KubeConfig:  kc,
			HubCluster:  r,
		}
		// ExternalId is an UUID.

		// TODO: if GKE cluster, try to determine real cluster name, location, project
		ep := r.GetEndpoint().GetGkeCluster()
		if ep != nil {
			// Format: //container.googleapis.com/projects/PID/locations/LOC/clusters/NAME
			parts := strings.Split(ep.ResourceLink, "/")
			if len(parts) == 9 && parts[2] == "container.googleapis.com" {
				c.ProjectId = parts[4]
				c.ClusterLocation = parts[6]
				c.ClusterName = parts[8]
			}
			log.Println("HUB:", parts)
		}

		ml = append(ml, c)

	}
	return ml, nil
}

func AllClusters(ctx context.Context, kr *mesh.KRun, configProjectId string,
	label string, meshID string) ([]*Cluster, error) {
	clustersL := []*Cluster{}

	cl, err := container.NewClusterManagerClient(ctx,option.WithQuotaProject(configProjectId))
	if err != nil {
		return nil, err
	}

	if configProjectId == "" {
		configProjectId = kr.Mesh.ProjectId
	}
	clcr := &containerpb.ListClustersRequest{
		Parent: "projects/" + configProjectId + "/locations/-",
	}
	clusters, err := cl.ListClusters(ctx, clcr)
	if err != nil {
		return nil, err
	}

	for _, c := range clusters.Clusters {
		if label != "" { // Filtered by label - if the filter is specified, ignore non-labeled clusters
			if meshID == "" { // If a value for label was specified, used it for filtering
				if c.ResourceLabels[label] == "" {
					continue
				}
			} else {
				if c.ResourceLabels[label] != meshID {
					continue
				}
			}
		}
		clustersL = append(clustersL, &Cluster{
			ProjectId:       configProjectId,
			ClusterName:     c.Name,
			ClusterLocation: c.Location,
			GKECluster:      c,
			KubeConfig:      addClusterConfig(c, configProjectId, c.Location, c.Name),
		})
	}
	return clustersL, nil
}

func addClusterConfig(c *containerpb.Cluster, p, l, clusterName string) *kubeconfig.Config {
	kc := kubeconfig.NewConfig()
	caCert, err := base64.StdEncoding.DecodeString(c.MasterAuth.ClusterCaCertificate)
	if err != nil {
		caCert = nil
	}

	ctxName := "gke_" + p + "_" + l + "_" + clusterName

	// We need a KUBECONFIG - tools/clientcmd/api/Config object
	kc.CurrentContext = ctxName
	kc.Contexts[ctxName] = &kubeconfig.Context{
		Cluster:  ctxName,
		AuthInfo: ctxName,
	}
	kc.Clusters[ctxName] = &kubeconfig.Cluster{
		Server:                   "https://" + c.Endpoint,
		CertificateAuthorityData: caCert,
	}
	kc.AuthInfos[ctxName] = &kubeconfig.AuthInfo{
		AuthProvider: &kubeconfig.AuthProviderConfig{
			Name: "gcp",
		},
	}
	kc.CurrentContext = ctxName

	return kc
}
