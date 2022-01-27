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
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"github.com/costinm/krun/k8s"
	kubeconfig "k8s.io/client-go/tools/clientcmd/api"
	// Required for k8s client to link in the authenticator
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	gkehub "cloud.google.com/go/gkehub/apiv1beta1"
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

//func ConfigGCP(ctx context.Context, kr *mesh.KRun) error {
//	// Avoid direct dependency on GCP libraries - may be replaced by a REST client or different XDS server discovery.
//	kc := &k8s.K8S{Mesh: kr, VendorInit: gcp.InitGCP}
//	err := kc.K8SClient(ctx)
//
//	return err
//}
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
