package uk8s

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"

	"golang.org/x/oauth2/google"
)

type Clusters struct {
	Clusters []*Cluster
}
type HubClusters struct {
	Resources []HubCluster
}

type HubCluster struct {
	// Full name - projects/wlhe-cr/locations/global/memberships/asm-cr
	//Name     string
	Endpoint *struct {
		GkeCluster *struct {
			// //container.googleapis.com/projects/wlhe-cr/locations/us-central1-c/clusters/asm-cr
			ResourceLink string
		}
		// kubernetesMetadata: vcpuCount, nodeCount, api version
	}
	State *struct {
		// READY
		Code string
	}

	Authority struct {
		Issuer               string `json:"issuer"`
		WorkloadIdentityPool string `json:"workloadIdentityPool"`
		IdentityProvider     string `json:"identityProvider"`
	} `json:"authority"`

	// Membership labels - different from GKE labels
	Labels map[string]string
}

type Cluster struct {
	Name string

	// nodeConfig
	MasterAuth struct {
		ClusterCaCertificate []byte
	}
	Location string

	Endpoint string

	ResourceLabels []string

	// Extras:

	// loggingService, monitoringService
	//Network string "default"
	//Subnetwork string
	ClusterIpv4Cidr string
	ServicesIpv4Cidr string
	// addonsConfig
	// nodePools

	// For regional clusters - each zone.
	// For zonal - one entry, equal with location
	Locations []string
	// ipAllocationPolicy - clusterIpv4Cider, serviceIpv4Cider...
	// masterAuthorizedNetworksConfig
	// maintenancePolicy
	// autoscaling
	NetworkConfig struct {
		// projects/NAME/global/networks/default
		Network string
		Subnetwork string
	}
	// releaseChannel
	// workloadIdentityConfig

	// It seems zone and location are the same for zonal clusters.
	//Zone string // ex: us-west1

}

func GetClusters(ctx context.Context, p string) ([]*Cluster, error) {
	httpClient, err := google.DefaultClient(ctx,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Get("https://container.googleapis.com/v1/projects/" + p + "/locations/-/clusters")
	log.Println(res.StatusCode)
	rd, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	//log.Println(string(rd))
	cl := &Clusters{}
	json.Unmarshal(rd, cl)

	return cl.Clusters, err
}

func GetCluster(ctx context.Context, path string) (*Cluster, error) {
	httpClient, err := google.DefaultClient(ctx,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Get("https://container.googleapis.com/v1" + path)
	log.Println(res.StatusCode)
	rd, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	//log.Println(string(rd))
	cl := &Cluster{}
	json.Unmarshal(rd, cl)

	return cl, err
}

func GetHubClusters(ctx context.Context, p string) ([]*Cluster, error) {
	httpClient, err := google.DefaultClient(ctx,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Get("https://gkehub.googleapis.com/v1/projects/" + p + "/locations/-/memberships")
	log.Println(res.StatusCode)
	rd, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	log.Println(string(rd))
	hcl := &HubClusters{}
	json.Unmarshal(rd, hcl)

	cl := []*Cluster{}
	for _, hc := range hcl.Resources {
		// hc doesn't provide the endpoint. Need to query GKE - but instead of going over each cluster we can do
		// batch query on the project and filter.
		if hc.Endpoint != nil && hc.Endpoint.GkeCluster != nil {
			ca := hc.Endpoint.GkeCluster.ResourceLink
			if strings.HasPrefix(ca, "//container.googleapis.com") {
				cc, err := GetCluster(ctx, ca[len("//container.googleapis.com"):])
				if err != nil {
					log.Println("Failed to get ", ca, err)
				}
				if cc != nil {
					cl = append(cl, cc)
				}
			}
		}

	}
	return cl, err
}
