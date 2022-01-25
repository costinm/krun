package urest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
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
	ClusterIpv4Cidr  string
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
		Network    string
		Subnetwork string
	}
	// releaseChannel
	// workloadIdentityConfig

	// It seems zone and location are the same for zonal clusters.
	//Zone string // ex: us-west1

}

// GKE2RestCluster will fill in the uk.Clusters with all clusters in the project.
// Will use an existing token (to avoid roundtrips for auth, access token can be shared)
func GKE2RestCluster(ctx context.Context, uk *UK8S, token string, p string) ([]*RestCluster, error) {
	req, _ := http.NewRequest("GET", "https://container.googleapis.com/v1/projects/"+p+"/locations/-/clusters", nil)
	req = req.WithContext(ctx)
	req.Header.Add("authorization", "Bearer "+token)

	res, err := uk.Client.Do(req)
	if res.StatusCode != 200 {
		rd, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		log.Println(string(rd))
		return nil, fmt.Errorf("Error reading clusters %d", res.StatusCode)
	}
	rd, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if Debug {
		log.Println(string(rd))
	}

	cl := &Clusters{}
	err = json.Unmarshal(rd, cl)
	if err != nil {
		return nil, err
	}
	rcl := []*RestCluster{}
	for _, c := range cl.Clusters {
		rc := &RestCluster{
			Client:        uk.httpClient(c.MasterAuth.ClusterCaCertificate),
			Base:          c.Endpoint,
			Location:      c.Location,
			TokenProvider: uk.TokenProvider,
			Id:            "gke_" + p + "_" + c.Location + "_" + c.Name,
		}
		rcl = append(rcl, rc)

		uk.add(rc)
	}

	return rcl, err
}

func (uk *UK8S) add(rc *RestCluster) {
	uk.m.Lock()
	if uk.Clusters[rc.Id] != nil {
		log.Println("Cluster in kube config")
	} else {
		uk.Clusters[rc.Id] = rc
		uk.ClustersByLocation[rc.Location] = append(uk.ClustersByLocation[rc.Location], rc)
	}
	uk.m.Unlock()
}

func GetCluster(ctx context.Context, uk *UK8S, token, path string) (*RestCluster, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://container.googleapis.com/v1"+path, nil)
	req.Header.Add("authorization", "Bearer "+token)

	parts := strings.Split(path, "/")
	p := parts[2]
	res, err := uk.Client.Do(req)
	log.Println(res.StatusCode)
	rd, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	c := &Cluster{}
	err = json.Unmarshal(rd, c)
	if err != nil {
		return nil, err
	}

	rc := &RestCluster{
		Client:        uk.httpClient(c.MasterAuth.ClusterCaCertificate),
		Base:          c.Endpoint,
		Location:      c.Location,
		TokenProvider: uk.TokenProvider,
		Id:            "gke_" + p + "_" + c.Location + "_" + c.Name,
	}

	return rc, err
}

func Hub2RestClusters(ctx context.Context, uk *UK8S, tok, p string) ([]*RestCluster, error) {
	req, _ := http.NewRequest("GET",
		"https://gkehub.googleapis.com/v1/projects/"+p+"/locations/-/memberships", nil)
	req = req.WithContext(ctx)
	req.Header.Add("authorization", "Bearer "+tok)

	res, err := uk.Client.Do(req)
	log.Println(res.StatusCode)
	rd, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	log.Println(string(rd))
	hcl := &HubClusters{}
	json.Unmarshal(rd, hcl)

	cl := []*RestCluster{}
	for _, hc := range hcl.Resources {
		// hc doesn't provide the endpoint. Need to query GKE - but instead of going over each cluster we can do
		// batch query on the project and filter.
		if hc.Endpoint != nil && hc.Endpoint.GkeCluster != nil {
			ca := hc.Endpoint.GkeCluster.ResourceLink
			if strings.HasPrefix(ca, "//container.googleapis.com") {
				rc, err := GetCluster(ctx, uk, tok, ca[len("//container.googleapis.com"):])
				if err != nil {
					log.Println("Failed to get ", ca, err)
				} else {
					uk.add(rc)
					cl = append(cl, rc)
				}
			}
		}
	}
	return cl, err
}

func queryMetadata(path string) (string, error) {
	// metadata.google.internal
	// TODO: read GCE_METADATA_HOST
	req, err := http.NewRequest("GET", "http://169.254.169.254/computeMetadata/v1/"+path, nil)
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