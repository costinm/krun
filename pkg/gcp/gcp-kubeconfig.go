package gcp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	container "cloud.google.com/go/container/apiv1"
	"github.com/costinm/krun/pkg/k8s"
	"k8s.io/client-go/kubernetes"

	gkehub "cloud.google.com/go/gkehub/apiv1beta1"
	gkehubpb "google.golang.org/genproto/googleapis/cloud/gkehub/v1beta1"

	crm "google.golang.org/api/cloudresourcemanager/v1"

	containerpb "google.golang.org/genproto/googleapis/container/v1"
	kubeconfig "k8s.io/client-go/tools/clientcmd/api"

	// Required for k8s client to link in the authenticator
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// TODO: if project/location not specified, get from local metadata server.
// TODO: if cluster not specified, list clusters
// TODO: use hub as well.

// Cluster wraps cluster information for a discovered hub or gke cluster.
type Cluster struct {
	ClusterName string
	ClusterLocation string
	ProjectId string

	GKECluster *containerpb.Cluster
	HubCluster *gkehubpb.Membership

	KubeConfig *kubeconfig.Config
}

var (
	GCPInitTime time.Duration
)

// configFromEnvAndMD will use env variables to locate the
// k8s cluster and create a client.
func configFromEnvAndMD(kr *k8s.KRun)  {
	if kr.ProjectId == "" {
		kr.ProjectId = os.Getenv("PROJECT_ID")
	}

	if kr.ClusterName == "" {
		kr.ClusterName = os.Getenv("CLUSTER_NAME")
	}

	if kr.ClusterLocation == "" {
		kr.ClusterLocation = os.Getenv("CLUSTER_LOCATION")
	}

	if kr.ProjectNumber == "" {
		kr.ProjectNumber = os.Getenv("PROJECT_NUMBER")
	}

	if os.Getenv("APPLICATION_DEFAULT_CREDENTIALS") == "" {
		// If ADC is set, we will only use the env variables. Else attempt to init from metadata server.
		if kr.ProjectId == "" {
			kr.ProjectId, _ = ProjectFromMetadata()
		}

		if kr.ClusterLocation == "" {
			kr.ClusterLocation, _ = RegionFromMetadata()
		}

		if kr.ProjectNumber == "" {
			kr.ProjectNumber, _ = ProjectNumberFromMetadata()
		}
		if kr.ProjectNumber == "" {
			kr.ProjectNumber = ProjectNumber(kr.ProjectId)
		}
	}

	if kr.ProjectNumber == "" && kr.ProjectId != "" {
		kr.ProjectNumber = ProjectNumber(kr.ProjectId)
	}
}

func RegionFromMetadata() (string, error) {
	v, err := queryMetadata("http://metadata.google.internal/computeMetadata/v1/instance/region")
	if err != nil {
		return "", err
	}
	vs := strings.SplitAfter(v, "/regions/")
	if len(vs) != 2 {
		return "", fmt.Errorf("malformed region value split into %#v", vs)
	}
	return vs[1], nil
}

func ProjectFromMetadata() (string, error) {
	v, err := queryMetadata("http://metadata.google.internal/computeMetadata/v1/project/project-id")
	if err != nil {
		return "", err
	}
	return v, nil
}

func ProjectNumberFromMetadata() (string, error) {
	v, err := queryMetadata("http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id")
	if err != nil {
		return "", err
	}
	return v, nil
}


func queryMetadata(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata server responeded with code=%d %s", resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), err
}


func InitGCP(ctx context.Context, kr *k8s.KRun) error {
	configFromEnvAndMD(kr)

	t0 := time.Now()
	var kc *kubeconfig.Config
	var err error

	if kr.ProjectId == "" {
		// GCP can't be initialized without a project ID
		return nil
	}

	if kr.ClusterName == "" || kr.ClusterLocation == "" {
		// ~500ms
		cl ,err := AllClusters(ctx, kr, "", "mesh_id", "")
		if err != nil {
			return err
		}

		if len(cl) == 0 {
			return nil // no cluster to use
		}

		kc = cl[0].KubeConfig
		// TODO: select default cluster based on location
		// WIP - list all clusters and attempt to find one in the same region.
		// TODO: connect to cluster, find istiod - and keep trying until a working
		// one is found ( fallback )

	} else {
		// ~400 ms
		cl, err := GKECluster(ctx, kr.ProjectId, kr.ClusterLocation, kr.ClusterName)
		//rc, err := CreateRestConfig(kr, kc, kr.ProjectId, kr.ClusterLocation, kr.ClusterName)
		if err != nil {
			return err
		}
		//kr.Client, err = kubernetes.NewForConfig(rc)
		kc = cl.KubeConfig
		if err != nil {
			log.Println("Failed in NewForConfig", kr, err)
			return err
		}
	}

	GCPInitTime = time.Since(t0)

	rc, err := restConfig(kc)
	if err != nil {
		return err
	}
	kr.Client, err = kubernetes.NewForConfig(rc)
	if err != nil {
		return err
	}

	SaveKubeConfig(kc, "./var/run/.kube", "config")

	return nil
}

func GKECluster(ctx context.Context, p, l, clusterName string) (*Cluster, error) {
	cl, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		log.Println("Failed NewClusterManagerClient", p, l, clusterName, err)
		return nil, err
	}

	for i :=0; i < 5; i++ {
		gcr := &containerpb.GetClusterRequest{
			Name: fmt.Sprintf("projects/%s/locations/%s/cluster/%s", p, l, clusterName),
		}
		c, e := cl.GetCluster(ctx, gcr)
		if e == nil {
			rc := &Cluster{
				ProjectId: p,
				ClusterLocation: c.Location,
				ClusterName: c.Name,
				GKECluster: c,
				KubeConfig: addClusterConfig(c, p, l, clusterName),
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
func AllHub(ctx context.Context, kr *k8s.KRun) ([]*Cluster, error) {
	cl, err := gkehub.NewGkeHubMembershipClient(ctx)
	if err != nil {
		return nil, err
	}

	mi := cl.ListMemberships(ctx, &gkehubpb.ListMembershipsRequest{
		Parent: "projects/" + kr.ProjectId + "/locations/-",
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
			break
		}
		mna := strings.Split(r.Name, "/")
		mn := mna[len(mna)-1]
		ctxName := "connectgateway_" + kr.ProjectId + "_"  + mn
		kc := kubeconfig.NewConfig()
		kc.Contexts[ctxName] = &kubeconfig.Context {
			Cluster:  ctxName,
			AuthInfo: ctxName,
		}
		kc.Clusters[ctxName] = &kubeconfig.Cluster {
			Server: fmt.Sprintf("https://connectgateway.googleapis.com/v1beta1/projects/%s/memberships/%s",
				kr.ProjectNumber, mn),
		}
		kc.AuthInfos[ctxName] = &kubeconfig.AuthInfo{
			AuthProvider: &kubeconfig.AuthProviderConfig{
				Name: "gcp",
			},
		}

		// TODO: better way to select default
		kc.CurrentContext = ctxName

		c := &Cluster {
			ProjectId: kr.ProjectId,
			ClusterName: r.Name,
			KubeConfig: kc,
			HubCluster: r,
		}
		// ExternalId is an UUID.

		// TODO: if GKE cluster, try to determine real cluster name, location, project
		ep :=	r.GetEndpoint()
		if ep != nil && ep.GkeCluster != nil {
			// Format: //container.googleapis.com/projects/PID/locations/LOC/clusters/NAME
			parts := strings.Split(ep.GkeCluster.ResourceLink, "/")
			if len(parts) == 9 && parts[2] == "container.googleapis.com" {
				c.ProjectId = parts[4]
				c.ClusterLocation = parts[6]
				c.ClusterName = parts[8]
			}
			log.Println(parts)
		}

		ml = append(ml, c)

	}
	return ml, nil
}


func AllClusters(ctx context.Context, kr *k8s.KRun, defCluster string, label string, meshID string) ([]*Cluster, error) {
	clustersL := []*Cluster{}

	if kr.ProjectId == "" {
		configFromEnvAndMD(kr)
	}
	if kr.ProjectId == "" {
		return nil, errors.New("requires PROJECT_ID")
	}

	cl, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return nil, err
	}

	clusters, err := cl.ListClusters(ctx, &containerpb.ListClustersRequest{
		Parent: "projects/" + kr.ProjectId + "/locations/-",
	})
	if err != nil {
		return nil,err
	}

	for _, c := range clusters.Clusters {
		if label != "" {
			if meshID == "" {
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
			ProjectId: kr.ProjectId,
			ClusterName: c.Name,
			ClusterLocation: c.Location,
			GKECluster: c,
			KubeConfig:		addClusterConfig(c, kr.ProjectId, c.Location, c.Name),

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
	kc.Contexts[ctxName]= &kubeconfig.Context {
		Cluster: ctxName,
		AuthInfo: ctxName,
	}
	kc.Clusters[ctxName] = &kubeconfig.Cluster{
		Server: "https://" + c.Endpoint,
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
