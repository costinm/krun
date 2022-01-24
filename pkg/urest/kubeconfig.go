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

package urest

import (
	"encoding/base64"
	"errors"
	"log"
	"strings"
)

// KubeConfig is the JSON representation of the kube config.
// The format supports most of the things we need and also allows connection to real k8s clusters.
// UGate implements a very light subset - should be sufficient to connect to K8S, but without any
// generated stubs. Based in part on https://github.com/ericchiang/k8s (abandoned), which is a light
// client.
type KubeConfig struct {
	// Must be v1
	ApiVersion string `json:"apiVersion"`
	// Must be Config
	Kind string `json:"kind"`

	// Clusters is a map of referencable names to cluster configs
	Clusters []*KubeNamedCluster `json:"clusters"`

	// AuthInfos is a map of referencable names to user configs
	Users []*KubeNamedUser `json:"users"`

	// Contexts is a map of referencable names to context configs
	Contexts []KubeNamedContext `json:"contexts"`

	// CurrentContext is the name of the context that you would like to use by default
	CurrentContext string `json:"current-context" yaml:"current-context"`
}

type KubeNamedCluster struct {
	Name    string      `json:"name"`
	Cluster KubeCluster `json:"cluster"`
}
type KubeNamedUser struct {
	Name string   `json:"name"`
	User KubeUser `json:"user"`
}
type KubeNamedContext struct {
	Name    string  `json:"name"`
	Context Context `json:"context"`
}

type KubeCluster struct {
	// LocationOfOrigin indicates where this object came from.  It is used for round tripping config post-merge, but never serialized.
	// +k8s:conversion-gen=false
	//LocationOfOrigin string
	// Server is the address of the kubernetes cluster (https://hostname:port).
	Server string `json:"server"`
	// InsecureSkipTLSVerify skips the validity check for the server's certificate. This will make your HTTPS connections insecure.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecure-skip-tls-verify,omitempty"`
	// CertificateAuthority is the path to a cert file for the certificate authority.
	// +optional
	CertificateAuthority string `json:"certificate-authority,omitempty" yaml:"certificate-authority"`
	// CertificateAuthorityData contains PEM-encoded certificate authority certificates. Overrides CertificateAuthority
	// +optional
	CertificateAuthorityData string `json:"certificate-authority-data,omitempty"  yaml:"certificate-authority-data"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	// +optional
	//Extensions map[string]runtime.Object `json:"extensions,omitempty"`
}

// KubeUser contains information that describes identity information.  This is use to tell the kubernetes cluster who you are.
type KubeUser struct {
	// LocationOfOrigin indicates where this object came from.  It is used for round tripping config post-merge, but never serialized.
	// +k8s:conversion-gen=false
	//LocationOfOrigin string
	// ClientCertificate is the path to a client cert file for TLS.
	// +optional
	ClientCertificate string `json:"client-certificate,omitempty"`
	// ClientCertificateData contains PEM-encoded data from a client cert file for TLS. Overrides ClientCertificate
	// +optional
	ClientCertificateData []byte `json:"client-certificate-data,omitempty"`
	// ClientKey is the path to a client key file for TLS.
	// +optional
	ClientKey string `json:"client-key,omitempty"`
	// ClientKeyData contains PEM-encoded data from a client key file for TLS. Overrides ClientKey
	// +optional
	ClientKeyData []byte `json:"client-key-data,omitempty"`
	// Token is the bearer token for authentication to the kubernetes cluster.
	// +optional
	Token string `json:"token,omitempty"`
	// TokenFile is a pointer to a file that contains a bearer token (as described above).  If both Token and TokenFile are present, Token takes precedence.
	// +optional
	TokenFile string `json:"tokenFile,omitempty"`
	// Impersonate is the username to act-as.
	// +optional
	//Impersonate string `json:"act-as,omitempty"`
	// ImpersonateGroups is the groups to imperonate.
	// +optional
	//ImpersonateGroups []string `json:"act-as-groups,omitempty"`
	// ImpersonateUserExtra contains additional information for impersonated user.
	// +optional
	//ImpersonateUserExtra map[string][]string `json:"act-as-user-extra,omitempty"`
	// Username is the username for basic authentication to the kubernetes cluster.
	// +optional
	Username string `json:"username,omitempty"`
	// Password is the password for basic authentication to the kubernetes cluster.
	// +optional
	Password string `json:"password,omitempty"`
	// AuthProvider specifies a custom authentication plugin for the kubernetes cluster.
	// +optional
	AuthProvider *struct {
		Name string `json:"name,omitempty"`
	} `json:"auth-provider,omitempty"`
	// Exec specifies a custom exec-based authentication plugin for the kubernetes cluster.
	// +optional
	//Exec *ExecConfig `json:"exec,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	// +optional
	//Extensions map[string]runtime.Object `json:"extensions,omitempty"`
}

// Context is a tuple of references to a cluster (how do I communicate with a kubernetes cluster), a user (how do I identify myself), and a namespace (what subset of resources do I want to work with)
type Context struct {
	// Cluster is the name of the cluster for this context
	Cluster string `json:"cluster"`
	// User is the name of the authInfo for this context
	User string `json:"user" yaml:"user"`
	// Namespace is the default namespace to use on unspecified requests
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

func kubeconfig2Rest(uk *UK8S, name string, cluster *KubeCluster, user *KubeUser, ns string) (*RestCluster, error) {
	rc := &RestCluster{
		Base:      cluster.Server,
		Token:     user.Token,
		Namespace: ns,
	}
	parts := strings.Split(name, "_")
	if parts[0] == "gke" {
		rc.ProjectId = parts[1]
		rc.Location = parts[2]
		rc.Name = parts[3]
	} else {
		rc.Name = name
	}
	rc.Id = name

	// May be useful to add: strings.HasPrefix(name, "gke_") ||
	if user.AuthProvider != nil && user.AuthProvider.Name == "gcp" {
		rc.TokenProvider = uk.TokenProvider
	}

	// TODO: support client cert, token file (with reload)
	caCert, err := base64.StdEncoding.DecodeString(cluster.CertificateAuthorityData)
	if err != nil {
		return nil, err
	}

	rc.Client = uk.httpClient(caCert)

	return rc, nil
}

func extractClusters(uk *UK8S, kc *KubeConfig) (*RestCluster, map[string][]*RestCluster, error) {
	var cluster *KubeCluster
	var user *KubeUser

	cByLoc := map[string][]*RestCluster{}
	cByName := map[string]*RestCluster{}

	if len(kc.Contexts) == 0 || kc.CurrentContext == "" {
		if len(kc.Clusters) == 0 || len(kc.Users) == 0 {
			return nil, cByLoc, errors.New("Kubeconfig has no clusters")
		}
		user = &kc.Users[0].User
		cluster = &kc.Clusters[0].Cluster
		rc, err := kubeconfig2Rest(uk, "", cluster, user, "")
		if err != nil {
			return nil, nil, err
		}
		cByLoc[rc.Location] = append(cByLoc[rc.Location], rc)
		return rc, cByLoc, nil
	}

	// Have contexts
	for _, cc := range kc.Contexts {
		for _, c := range kc.Clusters {
			if c.Name == cc.Context.Cluster {
				cluster = &c.Cluster
			}
		}
		for _, c := range kc.Users {
			if c.Name == cc.Context.User {
				user = &c.User
			}
		}
		rc, err := kubeconfig2Rest(uk, cc.Context.Cluster, cluster, user, cc.Context.Namespace)
		if err != nil {
			log.Println("Skipping incompatible cluster ", cc.Context.Cluster, err)
		} else {
			cByLoc[rc.Location] = append(cByLoc[rc.Location], rc)
			cByName[cc.Name] = rc
		}
	}

	if len(cByName) == 0 {
		return nil, nil, errors.New("no clusters found")
	}
	defc := cByName[kc.CurrentContext]
	if defc == nil {
		for _, c := range cByName {
			defc = c
			break
		}
	}
	return defc, cByLoc, nil
}
