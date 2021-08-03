package k8s

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	container "cloud.google.com/go/container/apiv1"

	gkehub "cloud.google.com/go/gkehub/apiv1beta1"
	//gkehub "google.golang.org/genproto/googleapis/cloud/gkehub/v1beta1"
	gkehubpb "google.golang.org/genproto/googleapis/cloud/gkehub/v1beta1"

	crm "google.golang.org/api/cloudresourcemanager/v1"

	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"k8s.io/client-go/rest"

	// Required for k8s client to link in the authenticator
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

)

// TODO: if project/location not specified, get from local metadata server.
// TODO: if cluster not specified, list clusters
// TODO: use hub as well.


// SaveKubeConfig saves the KUBECONFIG to ./var/run/.kube/config
// The assumption is that on a read-only image, /var/run will be
// writeable and not backed up.
func (kr *KRun) SaveKubeConfig() error {
	if kr.KubeConfig == nil {
		return nil
	}
	cfgjs, err := clientcmd.Write(*kr.KubeConfig)
	if err != nil {
		return err
	}
	err = os.MkdirAll("./var/run/.kube", 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("./var/run/.kube/config", cfgjs, 0744)
	if err != nil {
		return err
	}
	return nil
}

func NewKubeConfig() *clientcmdapi.Config {
	return &clientcmdapi.Config{
		APIVersion: "v1",
		Contexts: map[string]*clientcmdapi.Context{
		},
		Clusters: map[string]*clientcmdapi.Cluster{
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
		},
	}
}

// CreateClusterConfig adds a cluster to the config.
func (kr *KRun) CreateClusterConfig(p, l, clusterName string) error {
	ctx := context.Background()

	cl, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return err
	}

	c, err := cl.GetCluster(ctx, &containerpb.GetClusterRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/cluster/%s", p, l, clusterName),
	})
	if err != nil {
		return err
	}
	return kr.addClusterConfig(c, p, l, clusterName)
}

func (kr *KRun) addClusterConfig(c *containerpb.Cluster, p, l, clusterName string) error {
	if kr.KubeConfig == nil {
		kr.KubeConfig = NewKubeConfig()
	}
	caCert, err := base64.StdEncoding.DecodeString(c.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return err
	}

	ctxName := "gke_" + p + "_" + l + "_" + clusterName

	// We need a KUBECONFIG - tools/clientcmd/api/Config object
	kr.KubeConfig.CurrentContext = ctxName
	kr.KubeConfig.Contexts[ctxName]= &clientcmdapi.Context {
				Cluster: ctxName,
				AuthInfo: ctxName,
	}
	kr.KubeConfig.Clusters[ctxName] = &clientcmdapi.Cluster{
				Server: "https://" + c.Endpoint,
				CertificateAuthorityData: caCert,
	}
	kr.KubeConfig.AuthInfos[ctxName] = &clientcmdapi.AuthInfo{
				AuthProvider: &clientcmdapi.AuthProviderConfig{
					Name: "gcp",
				},
	}

	return nil
}

// CreateRestConfig will create a k8s client for the project, location and cluster
//
// If cluster name is missing, will list projects in the same location, and pick
// the first project with mesh_id label.
func (kr *KRun) CreateRestConfig(p, l, clusterName string) (*rest.Config, error) {
	ctx := context.Background()

	cl, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		log.Println("Failed NewClusterManagerClient", kr, err)
		return nil, err
	}

	for i :=0; i < 5; i++ {
		gcr := &containerpb.GetClusterRequest{
			Name: fmt.Sprintf("projects/%s/locations/%s/cluster/%s", p, l, clusterName),
		}
		c, e := cl.GetCluster(ctx, gcr)
		if e == nil {
			kr.Clusters = append(kr.Clusters, c)

			kr.addClusterConfig(c, p, l, clusterName)
			return kr.restConfigForCluster(c)
		}
		log.Println("Failed GetCluster, retry", gcr, kr, err)
		time.Sleep(1 * time.Second)
		err = e
	}
	return nil, err
}

func (kr *KRun) restConfigForCluster(c *containerpb.Cluster) (*rest.Config, error) {
	caCert, err := base64.StdEncoding.DecodeString(c.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, err
	}

	// This is a rest.Config - can be used directly with the rest API
	cfg := &rest.Config{
		Host: "https://" + c.Endpoint,
		AuthProvider: &clientcmdapi.AuthProviderConfig{
			Name: "gcp",
		},
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caCert,
		},
	}

	cfg.TLSClientConfig.CAData = caCert

	return cfg, nil
}

func ProjectNumber(p string) string {
	ctx := context.Background()

	cr, err := crm.NewService(ctx)
	pdata, err := cr.Projects.Get(p).Do()
	if err != nil {
		log.Println("Error getting project number", p, err)
		return p
	}

	// This is in v1 - v3 has it encoded in name.
	return strconv.Itoa(int(pdata.ProjectNumber))
}

func (kr *KRun) AllHub(project string, defCluster string, label string, meshID string) error {
	ctx := context.Background()
	if kr.KubeConfig == nil {
		kr.KubeConfig = NewKubeConfig()
	}

	cl, err := gkehub.NewGkeHubMembershipClient(ctx)
	if err != nil {
		return err
	}

	//cl.GenerateConnectManifest()
	mi := cl.ListMemberships(ctx, &gkehubpb.ListMembershipsRequest{
		Parent: "projects/" + project + "/locations/-",
	})
	pn := ProjectNumber(project)
	for {
		r, err := mi.Next()
		//fmt.Println(r, err)
		if err != nil || r == nil {
			break
		}
		if label != "ALL" {
			if r.Labels[label] != meshID {
				continue
			}
		}

		mna := strings.Split(r.Name, "/")
		mn := mna[len(mna)-1]
		ctxName := "connectgateway_" + project + "_"  + mn
		kr.KubeConfig.Contexts[ctxName] = &clientcmdapi.Context {
			Cluster:  ctxName,
			AuthInfo: ctxName,
		}
		kr.KubeConfig.Clusters[ctxName] = &clientcmdapi.Cluster {
			Server: fmt.Sprintf("https://connectgateway.googleapis.com/v1beta1/projects/%s/memberships/%s",
				pn, mn),
		}
		kr.KubeConfig.AuthInfos[ctxName] = &clientcmdapi.AuthInfo{
			AuthProvider: &clientcmdapi.AuthProviderConfig{
				Name: "gcp",
			},
		}

		if mn == defCluster {
			kr.KubeConfig.CurrentContext = ctxName
		}

	}
	return nil
}

func (kr *KRun) AllClusters(project string, defCluster string, label string, meshID string) error {
	ctx := context.Background()
	if kr.KubeConfig == nil {
		kr.KubeConfig = NewKubeConfig()
	}

	cl, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return err
	}

	clusters, err := cl.ListClusters(ctx, &containerpb.ListClustersRequest{
		Parent: "projects/" + project + "/locations/-",
	})
	if err != nil {
		return err
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
		kr.Clusters = append(kr.Clusters, c)

		caCert, err := base64.StdEncoding.DecodeString(c.MasterAuth.ClusterCaCertificate)
		if err != nil {
			return err
		}

		// This is a rest.Config - can be used directly with the rest API
		cfg := &rest.Config{
			Host: "https://" + c.Endpoint,
			AuthProvider: &clientcmdapi.AuthProviderConfig{
				Name: "gcp",
				Config: map[string]string{},
			},
			TLSClientConfig: rest.TLSClientConfig{
				CAData: caCert,
			},
		}

		cfg.TLSClientConfig.CAData = caCert

		ctxName := "gke_" + project + "_" + c.Location + "_" + c.Name

		// We need a KUBECONFIG - tools/clientcmd/api/Config object
		kr.KubeConfig.Contexts[ctxName] = &clientcmdapi.Context {
					Cluster:  ctxName,
					AuthInfo: ctxName,
				}
		kr.KubeConfig.Clusters[ctxName] = &clientcmdapi.Cluster {
					Server:                   "https://" + c.Endpoint,
					CertificateAuthorityData: caCert,
				}
		kr.KubeConfig.AuthInfos[ctxName] = &clientcmdapi.AuthInfo{
			AuthProvider: &clientcmdapi.AuthProviderConfig{
				Name: "gcp",
			},
		}
		if c.Name == defCluster {
			kr.KubeConfig.CurrentContext = ctxName
		}
	}
	return nil
}
