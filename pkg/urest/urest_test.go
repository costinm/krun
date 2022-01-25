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

package urest_test

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/costinm/krun/pkg/mesh"
	"github.com/costinm/krun/pkg/urest"
	"golang.org/x/oauth2/google"
	"gopkg.in/yaml.v2"
)


var (
	hubResponse = `
`

	// This is a real GKE cluster list response, for unit testing and as example.
	// This package is only using a small subset of the returned information.
	gkeResponse = `
{
  "clusters": [
    {
      "name": "istio",
      "nodeConfig": {
        "machineType": "e2-medium",
        "diskSizeGb": 100,
        "oauthScopes": [
          "https://www.googleapis.com/auth/devstorage.read_only",
          "https://www.googleapis.com/auth/logging.write",
          "https://www.googleapis.com/auth/monitoring",
          "https://www.googleapis.com/auth/service.management.readonly",
          "https://www.googleapis.com/auth/servicecontrol",
          "https://www.googleapis.com/auth/trace.append"
        ],
        "metadata": {
          "disable-legacy-endpoints": "true"
        },
        "imageType": "COS_CONTAINERD",
        "serviceAccount": "default",
        "diskType": "pd-standard",
        "workloadMetadataConfig": {
          "mode": "GKE_METADATA"
        },
        "shieldedInstanceConfig": {
          "enableSecureBoot": true,
          "enableIntegrityMonitoring": true
        }
      },
      "masterAuth": {
        "clusterCaCertificate": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVMVENDQXBXZ0F3SUJBZ0lSQU1ybVVQd1R6cVhWUitJck92dnlsaG93RFFZSktvWklodmNOQVFFTEJRQXcKTHpFdE1Dc0dBMVVFQXhNa01EZ3daV0ppT0RJdE56ZzVaaTAwWlRCa0xXSmxPREV0TURRNVptWTRPVE5tTmpBMgpNQ0FYRFRJeE1UQXhOREl5TURVMU5Gb1lEekl3TlRFeE1EQTNNak13TlRVMFdqQXZNUzB3S3dZRFZRUURFeVF3Ck9EQmxZbUk0TWkwM09EbG1MVFJsTUdRdFltVTRNUzB3TkRsbVpqZzVNMlkyTURZd2dnR2lNQTBHQ1NxR1NJYjMKRFFFQkFRVUFBNElCandBd2dnR0tBb0lCZ1FEWmxkRzJvVGMzNlNFWjIyNk1YdkltNUgvZ2Q2RUxOc2xGaEQycgpjMXJDL3ZFMVNkbjlhV1dnUElUK0NPL1hDZkp1NTR4VUpFRFpsMmVKcHdCUkhVWDlJUmlFVUVXVkFoQjYwaG50CkhUM0t0RThEMVFQV2hYa1h6clNUUGJOWkhmYnRvTDJWVkZUKzFMTFdFUEdHZzBPc1VRcVZzYytMaE1LaytxYnUKN2k5c1NpcGVxZ3ozR2VvLzRkMFR3ajhUMGc1ZTVrR04vb1paN1l1bkxBWnVsd044QlJLREFwTzJTYXdmdU9QcgpEVEdIVEx1YTBoQzEvc1NnNnhOdlMxbnF0UmQwdzhDWG5vMmlJbGlXSE16WStZbGhkbWc2KzNaeEtrVGlnaHhjCjFSQ2JpSkoyOTdieGtQMlZib1NtOGE5UEk1QnZMY3M0N2E5QXRUMHQ0WEJDRkxQb1oxeFZWRzNFN01haVd0ZE4KaEFtRlNLWjlFa2l1WUIrT2Nobkkyek1qalI2ZXJIZkxUazZJbzAzVlVIVzlSbVI1RlZmbjNzSmFmTUVsZ2FuSQpWUXduRjZ2UXM0Z3pXOVdFa0prU2tRT3BoS2FhUXVrWlA3VWhERGc0SkVUbis2Ti9qVlNXS2ZTTEd1YVZOY2NECndHVXNTc2Jrb3ZUZ2FqWjlscEI4cUNXdmpnTUNBd0VBQWFOQ01FQXdEZ1lEVlIwUEFRSC9CQVFEQWdJRU1BOEcKQTFVZEV3RUIvd1FGTUFNQkFmOHdIUVlEVlIwT0JCWUVGSGtNbXZFdExxWDN0eFl1dkxURytleG5UcDZ6TUEwRwpDU3FHU0liM0RRRUJDd1VBQTRJQmdRQVNIYXluNXVNbEdNeE96YW1nZmYwcncxaS9uTzJvb0paa0VIajJ4NWViCjdaNVFRWm9pY2FrdDVlSk1MR3g4ZlVLVTQ5WC9kZnZRd204Z0hiNUlxcVFXSW9ITEhoYldCVDkrT2xaT085LzYKVTR0ejU5Rk8zcytzOUVONlVFZ2dzQlhPS05nNGpaNHhLQTJpelVmd0E4ckVDU2ZIenduRXZPWXVOaGV6MzlZYwpLLzZMSmd2Y09WbGhZaTM1enNhcDhOcmpkeVozVXF4ZFpuSXNGaHhhdTdWWjQvdHNQWFZKbnVjcU1LR1gvMjhVCkMxK2s1NU5QSk1Pa0V1R2dmL0xUckF1NFU1RVRXZG9iNXBhS0QxTVZIMkNlMkNacWtrTmxUN0FwWWpuU0VTRncKZEhvbzRBZW1JOUE4clVHZWNibU9ZUUhrWGhFUC91Y0Z4NmdTMDN4bktpYVdUTkx6VHE0b1V5VzlwemhaUEpOOApOZDZ4dFdnVXQ0UVB6TE5PTm4rMk4zaFQ0UURVditPMnBjVmtadHZvZzBFeEZ0ZDZ6ZDA5c1N1V2ovUE9WQnQ0ClBEU3JrUGZ3WmVtaWR1MjVNamxpVWV1NDJrSjlRZVdvRmdNaVQvY1ZLTVh3QUo5bU8va2JEY3lPWmRQUmp6cEMKNmVYb09iUGdiTUswZUhuY0hFeW43RW89Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
      },
      "loggingService": "logging.googleapis.com/kubernetes",
      "monitoringService": "monitoring.googleapis.com/kubernetes",
      "network": "default",
      "clusterIpv4Cidr": "10.48.128.0/17",
      "addonsConfig": {
        "httpLoadBalancing": {},
        "horizontalPodAutoscaling": {},
        "kubernetesDashboard": {
          "disabled": true
        },
        "networkPolicyConfig": {
          "disabled": true
        },
        "dnsCacheConfig": {
          "enabled": true
        },
        "gcePersistentDiskCsiDriverConfig": {
          "enabled": true
        }
      },
      "subnetwork": "default",
      "nodePools": [
        {
          "name": "default-pool",
          "config": {
            "machineType": "e2-medium",
            "diskSizeGb": 100,
            "oauthScopes": [
              "https://www.googleapis.com/auth/devstorage.read_only",
              "https://www.googleapis.com/auth/logging.write",
              "https://www.googleapis.com/auth/monitoring",
              "https://www.googleapis.com/auth/service.management.readonly",
              "https://www.googleapis.com/auth/servicecontrol",
              "https://www.googleapis.com/auth/trace.append"
            ],
            "metadata": {
              "disable-legacy-endpoints": "true"
            },
            "imageType": "COS_CONTAINERD",
            "serviceAccount": "default",
            "diskType": "pd-standard",
            "workloadMetadataConfig": {
              "mode": "GKE_METADATA"
            },
            "shieldedInstanceConfig": {
              "enableSecureBoot": true,
              "enableIntegrityMonitoring": true
            }
          },
          "initialNodeCount": 1,
          "autoscaling": {
            "enabled": true,
            "maxNodeCount": 1000,
            "autoprovisioned": true
          },
          "management": {
            "autoUpgrade": true,
            "autoRepair": true
          },
          "maxPodsConstraint": {
            "maxPodsPerNode": "32"
          },
          "podIpv4CidrSize": 26,
          "locations": [
            "us-west1-b",
            "us-west1-a"
          ],
          "networkConfig": {
            "podRange": "gke-istio-pods-290b29a3",
            "podIpv4CidrBlock": "10.48.128.0/17"
          },
          "selfLink": "https://container.googleapis.com/v1/projects/wlhe-cr/locations/us-west1/clusters/istio/nodePools/default-pool",
          "version": "1.20.10-gke.1600",
          "instanceGroupUrls": [
            "https://www.googleapis.com/compute/v1/projects/wlhe-cr/zones/us-west1-b/instanceGroupManagers/gk3-istio-default-pool-a13bd571-grp",
            "https://www.googleapis.com/compute/v1/projects/wlhe-cr/zones/us-west1-a/instanceGroupManagers/gk3-istio-default-pool-86024aa2-grp"
          ],
          "status": "RUNNING",
          "upgradeSettings": {
            "maxSurge": 1
          }
        }
      ],
      "locations": [
        "us-west1-a",
        "us-west1-b",
        "us-west1-c"
      ],
			"resourceLabels": {
        "istio": "config"
      },
      "labelFingerprint": "a9dc16a7",
      "legacyAbac": {},
      "ipAllocationPolicy": {
        "useIpAliases": true,
        "clusterIpv4Cidr": "10.48.128.0/17",
        "servicesIpv4Cidr": "10.48.16.0/22",
        "clusterSecondaryRangeName": "gke-istio-pods-290b29a3",
        "servicesSecondaryRangeName": "gke-istio-services-290b29a3",
        "clusterIpv4CidrBlock": "10.48.128.0/17",
        "servicesIpv4CidrBlock": "10.48.16.0/22"
      },
      "masterAuthorizedNetworksConfig": {},
      "maintenancePolicy": {
        "resourceVersion": "e3b0c442"
      },
      "autoscaling": {
        "enableNodeAutoprovisioning": true,
        "resourceLimits": [
          {
            "resourceType": "cpu",
            "maximum": "1000000000"
          },
          {
            "resourceType": "memory",
            "maximum": "1000000000"
          }
        ],
        "autoscalingProfile": "OPTIMIZE_UTILIZATION",
        "autoprovisioningNodePoolDefaults": {
          "oauthScopes": [
            "https://www.googleapis.com/auth/devstorage.read_only",
            "https://www.googleapis.com/auth/logging.write",
            "https://www.googleapis.com/auth/monitoring",
            "https://www.googleapis.com/auth/service.management.readonly",
            "https://www.googleapis.com/auth/servicecontrol",
            "https://www.googleapis.com/auth/trace.append"
          ],
          "serviceAccount": "default",
          "upgradeSettings": {
            "maxSurge": 1
          },
          "management": {
            "autoUpgrade": true,
            "autoRepair": true
          },
          "imageType": "COS_CONTAINERD"
        }
      },
      "networkConfig": {
        "network": "projects/wlhe-cr/global/networks/default",
        "subnetwork": "projects/wlhe-cr/regions/us-west1/subnetworks/default",
        "enableIntraNodeVisibility": true,
        "defaultSnatStatus": {}
      },
      "defaultMaxPodsConstraint": {
        "maxPodsPerNode": "110"
      },
      "databaseEncryption": {
        "state": "DECRYPTED"
      },
      "verticalPodAutoscaling": {
        "enabled": true
      },
      "shieldedNodes": {
        "enabled": true
      },
      "releaseChannel": {
        "channel": "REGULAR"
      },
      "workloadIdentityConfig": {
        "workloadPool": "wlhe-cr.svc.id.goog"
      },
      "notificationConfig": {
        "pubsub": {}
      },
      "selfLink": "https://container.googleapis.com/v1/projects/wlhe-cr/locations/us-west1/clusters/istio",
      "zone": "us-west1",
      "endpoint": "34.83.148.180",
      "initialClusterVersion": "1.20.10-gke.301",
      "currentMasterVersion": "1.20.10-gke.1600",
      "currentNodeVersion": "1.20.10-gke.1600",
      "createTime": "2021-10-14T23:05:54+00:00",
      "status": "RUNNING",
      "servicesIpv4Cidr": "10.48.16.0/22",
      "instanceGroupUrls": [
        "https://www.googleapis.com/compute/v1/projects/wlhe-cr/zones/us-west1-b/instanceGroupManagers/gk3-istio-default-pool-a13bd571-grp",
        "https://www.googleapis.com/compute/v1/projects/wlhe-cr/zones/us-west1-a/instanceGroupManagers/gk3-istio-default-pool-86024aa2-grp"
      ],
      "currentNodeCount": 6,
      "location": "us-west1",
      "autopilot": {
        "enabled": true
      },
      "id": "290b29a3fba948319b856202b60f8985209bd9f7c6ad409f97458adeea0cd336",
      "nodePoolDefaults": {
        "nodeConfigDefaults": {}
      },
      "loggingConfig": {
        "componentConfig": {
          "enableComponents": [
            "SYSTEM_COMPONENTS",
            "WORKLOADS"
          ]
        }
      },
      "monitoringConfig": {
        "componentConfig": {
          "enableComponents": [
            "SYSTEM_COMPONENTS"
          ]
        }
      }
    }
	]
}
`
)

// Requires a GSA (either via GOOGLE_APPLICATION_CREDENTIALS, gcloud config, metadata) with hub and
// container access.
// Requires a kube config - the default cluster should be in same project.
//
// Will verify kube config loading and queries to hub and gke.
//
func TestURest(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	defer cf()

	uk := urest.New()
	ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		t.Fatal(err)
	}
	uk.TokenProvider = func(ctx context.Context, s string) (string, error) {
		t, err := ts.Token()
		if err != nil {
			return "", err
		}
		return t.AccessToken, nil
	}
	kcd, err := ioutil.ReadFile(os.Getenv("HOME") + "/.kube/config")
	if err != nil {
		t.Skip("No k8s config", err)
	}
	kc := &urest.KubeConfig{}
	err = yaml.Unmarshal(kcd, kc)
	if err != nil {
		t.Fatal("Failed to parse k8c", err)
	}
	if kc.CurrentContext == "" {
		t.Fatal("No default context", kc)
	}
	uk.initDefaultTokenSource(ctx)
	uk.Client = uk.httpClient(nil)

	ecl, _, err := extractClusters(uk, kc)
	if err != nil {
		t.Fatal("Failed to load k8s", err)
	}

	//
	uk.Current = ecl
	uk.ProjectID = ecl.ProjectId

	// Cluster must have mesh connector installed
	t.Run("kubeconfig", func(t *testing.T) {
		cm, err := uk.GetCM(ctx, "istio-system", "mesh-env")
		if err != nil {
			t.Fatal("Failed to load k8s", err)
		}
		log.Println(cm)
	})

	// Access tokens
	tok, err := uk.TokenProvider(ctx, "")

	t.Run("hublist", func(t *testing.T) {
		cd, err := urest.Hub2RestClusters(ctx, uk, tok, uk.ProjectID)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cd {
			//uk.Current = c

			log.Println(c)
		}
	})

	t.Run("gkelist", func(t *testing.T) {
		cd, err := urest.GKE2RestCluster(ctx, uk, tok, uk.ProjectID)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cd {
			log.Println(c)
		}
	})

	t.Run("secret", func(t *testing.T) {
		cd, err := urest.GcpSecret(ctx, uk, tok, uk.ProjectID, "ca", "1")
		if err != nil {
			t.Fatal(err)
		}
		log.Println(string(cd))
	})

	t.Run("init", func(t *testing.T) {
		kr := mesh.New("")
		kr.ProjectId = uk.ProjectID

		uk1, err := urest.K8SClient(ctx, kr)
		uk1.Client = http.DefaultClient

		cd, err := urest.GKE2RestCluster(context.TODO(), uk1, uk.ProjectID, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cd {
			log.Println(c)
		}
	})

}
