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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/costinm/krun/pkg/mesh"
	"golang.org/x/oauth2/google"
	"gopkg.in/yaml.v2"
)

// WIP - removing the dep on k8s client library for the 2 bootstrap requests needed
// (get mesh-env and tokens).

// uK8S is a micro client for K8S, intended for bootstraping and minimal
// config.
///
// uK8S implements read access to K8S API servier using http requests
// It only supports GET - primarily to download the mesh-env config map,
// and the TokenRequest API needed for tokens/certificates.

// Based on/inspired from kelseyhightower/konfig

//
// It uses JWT tokens for auth, with the default credentials, and supports
// basic resources used for bootstrap. It is not intended for watching resources
// or complicated operations (list, write), only quick get of few configmaps, secret, services
//
// Refactored/extacted from 'konfig'

var (
	projectName    = "konfig"
	projectVersion = "0.1.0"
	projectURL     = "https://github.com/costinm/konfig"
	userAgent      = fmt.Sprintf("%s/%s (+%s; %s)",
		projectName, projectVersion, projectURL, runtime.Version())
)

// UK8S is a micro k8s client, using only base http and a token source.
type UK8S struct {
	// Client using system certificates, for external sites.
	// The RestCluster has a separate Client, configured to authenticate servers using a custom root CA.
	Client        *http.Client

	TokenProvider func(context.Context, string) (string, error)

	// Hub or user project ID. If set, will be used to lookup clusters.
	ProjectID string

	// Location where the workload is running, to select local clusters.
	Location string

	// Configs about the mesh.
	Mesh *mesh.KRun

	// Current active cluster.
	Current *RestCluster

	Clusters           map[string]*RestCluster
	ClustersByLocation map[string][]*RestCluster

	m sync.RWMutex

	TransportWrapper func(transport http.RoundTripper) http.RoundTripper
}

type RestCluster struct {
	// Base URL, including https://IP:port/v1
	// Created from endpoint.
	Base string

	// Token is set if the project is created from a k8s config using
	// the long lived secret (for example Istio)
	Token string
	// Optional TokenProvider - not needed if client wraps google oauth.
	TokenProvider func(context.Context, string) (string, error)

	// Cluster ID - the cluster name in kube config, hub, gke
	Id string

	Name      string
	Location  string
	ProjectId string

	// Default namespace - extracted from kubeconfig
	Namespace string
	Client    *http.Client
}

func (uK8S *UK8S) String() string {
	return uK8S.Current.Id
}

func (uK8S *UK8S) httpClient(caCert []byte) *http.Client {
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caCert) {
		log.Println("Failed to decode PEM")
	}

	// The 'max idle conns, idle con timeout, etc are shorter - this is meant for
	// fast initial config, not as a general purpose client.
	var tr http.RoundTripper
	tr = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{
			RootCAs: roots,
		},
	}

	if uK8S.TransportWrapper != nil {
		tr = uK8S.TransportWrapper(tr)
	}

	return &http.Client{
		Transport: tr,
	}

}

var Debug = false

func (uk8s *UK8S) Do(ctx context.Context, ns, kind, name string, postdata []byte) ([]byte, error) {

	resourceURL := uk8s.Current.Base + fmt.Sprintf("/api/v1/namespaces/%s/%ss/%s",
		ns, kind, name)

	var resp *http.Response
	var err error
	var req *http.Request
	if postdata == nil {
		req, _ = http.NewRequest("GET", resourceURL, nil)
	} else {
		req, _ = http.NewRequest("POST", resourceURL, bytes.NewReader(postdata))
		req.Header.Add("content-type", "application/json")
	}

	req = req.WithContext(ctx)

	if uk8s.Current.Token != "" {
		req.Header.Add("authorization", "bearer "+uk8s.Current.Token)
	} else if uk8s.Current.TokenProvider != nil {
		t, err := uk8s.Current.TokenProvider(ctx, uk8s.Current.Base)
		if err != nil {
			return nil, err
		}
		req.Header.Add("authorization", "bearer "+t)
	}

	resp, err = uk8s.Current.Client.Do(req)
	if Debug {
		log.Println(req, resp, err)
	}

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return nil, errors.New(fmt.Sprintf("kconfig: unable to get %s.%s %s from Kubernetes status code %v",
			ns, name, kind, resp.StatusCode))
	}

	return data, nil
}

func New() *UK8S {
	return &UK8S{
		ClustersByLocation: map[string][]*RestCluster{},
		Clusters: map[string]*RestCluster{},
		Client: http.DefaultClient,
	}
}

// Init the K8S client:
//
// 1. Explicit KUBECONFIG
// 2. GCP APIs, selecting a cluster.
//
func K8SClient(ctx context.Context, m *mesh.KRun) (*UK8S, error) {
	uk := New()
	uk.Mesh = m
	uk.TransportWrapper = m.TransportWrapper
	m.Cfg = uk
	uk.Client = http.DefaultClient
	m.TokenProvider = uk

	// Init GCP auth
	// DefaultTokenSource will:
	// - check GOOGLE_APPLICATION_CREDENTIALS
	// - ~/.config/gcloud/application_default_credentials.json"
	// - use metadata
	ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}
	uk.TokenProvider = func(ctx context.Context, s string) (string, error) {
		t, err := ts.Token()
		if err != nil {
			return "", err
		}
		return t.AccessToken, nil
	}
	kc := os.Getenv("KUBECONFIG")
	if kc == "" {
		kc = os.Getenv("HOME") + "/.kube/config"
	}
	if kc != "" {
		if _, err := os.Stat(kc); err == nil {
			// Explicit kube config, using it.
			kcd, err := ioutil.ReadFile(kc)
			if err != nil {
				return nil, err
			}
			kc := &KubeConfig{}
			err = yaml.Unmarshal(kcd, kc)
			if err != nil {
				return nil, err
			}

			def, cbyl, err := KubeConfig2RestCluster(uk, kc)
			if err != nil {
				return nil, err
			}

			uk.Current = def
			uk.ClustersByLocation = cbyl
		}
	}

	// Using GCP APIs
	if m.ProjectId != "" {
		accessToken, err := uk.TokenProvider(ctx, "")
		if err != nil {
			log.Println("Failed to load GKE clusters, no token", err)
		}

		_, err = GKE2RestCluster(ctx, uk, accessToken, m.ProjectId)
		if err != nil {
			return nil, err
		}
	}

	// TODO: select the config clusters
	// TODO: function to update the 'active' cluster on failure, locate local one

	return uk, nil
}