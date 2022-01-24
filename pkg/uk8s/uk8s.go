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

package uk8s

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
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

// Copyright 2019 The Konfig Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

type Secret struct {
	ApiVersion string            `json:"apiVersion"`
	Data       map[string][]byte `json:"data"`
	Kind       string            `json:"kind"`
}

type ConfigMap struct {
	ApiVersion string            `json:"apiVersion"`
	Data       map[string]string `json:"data"`
	Kind       string            `json:"kind"`
}

type CreateTokenResponseStatus struct {
	Token string `json:"token"`
}

type CreateTokenRequestSpec struct {
	Audiences []string `json:"audiences"`
}

type CreateTokenRequest struct {
	Spec CreateTokenRequestSpec `json:"spec"`
}

type CreateTokenResponse struct {
	Status CreateTokenResponseStatus `json:"status"`
}

var (
	projectName    = "konfig"
	projectVersion = "0.1.0"
	projectURL     = "https://github.com/costinm/konfig"
	userAgent      = fmt.Sprintf("%s/%s (+%s; %s)",
		projectName, projectVersion, projectURL, runtime.Version())
)

// UK8S is a micro k8s client, using only base http and a token source.
type UK8S struct {
	// Client using platform tokens and system certs
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

	// If TransportWrapper is set, the http clients will be wrapped
	// This is intended for integration with OpenTelemetry or other transport wrappers.
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

	// Client configured with the root CA of the K8S cluster.
	Client    *http.Client
}

func (uK8S *UK8S) String() string {
	return uK8S.Current.Id
}

// httpClient returns a http.Client configured with the specified root CA.
func (uK8S *UK8S) httpClient(caCert []byte) *http.Client {
	// The 'max idle conns, idle con timeout, etc are shorter - this is meant for
	// fast initial config, not as a general purpose client.
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	}
	if caCert != nil && len(caCert) > 0 {
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(caCert) {
			log.Println("Failed to decode PEM")
		}
		tr.TLSClientConfig = &tls.Config{
			RootCAs: roots,
		}
	}

	var rt http.RoundTripper
	rt = tr
	if uK8S.TransportWrapper != nil {
		rt = uK8S.TransportWrapper(rt)
	}

	return &http.Client{
		Transport: rt,
	}

}

func (uK8S *UK8S) GetSecret(ctx context.Context, ns string, name string) (map[string][]byte, error) {
	data, err := uK8S.Do(ctx, ns, "secret", name, nil)
	if err != nil {
		return nil, err
	}
	var secret Secret
	err = json.Unmarshal(data, &secret)
	if err != nil {
		return nil, err
	}

	return secret.Data, nil
}

func (uK8S *UK8S) GetCM(ctx context.Context, ns string, name string) (map[string]string, error) {
	data, err := uK8S.Do(ctx, ns, "configmap", name, nil)
	if err != nil {
		return nil, err
	}
	var secret ConfigMap
	err = json.Unmarshal(data, &secret)
	if err != nil {
		return nil, err
	}

	return secret.Data, nil
}

func (uK8S *UK8S) GetToken(ctx context.Context, aud string) (string, error) {
	return uK8S.GetTokenRaw(ctx, uK8S.Mesh.Namespace, uK8S.Mesh.Name, aud)
}

func (uK8S *UK8S) GetTokenRaw(ctx context.Context, ns, name, aud string) (string, error) {
	// If no audience is specified, something like
	//   https://container.googleapis.com/v1/projects/costin-asm1/locations/us-central1-c/clusters/big1
	// is generated ( on GKE ) - which seems to be the audience for K8S
	data, err := uK8S.Do(ctx, ns, "serviceaccount", name+"/token", []byte(fmt.Sprintf(`
{"kind":"TokenRequest","apiVersion":"authentication.k8s.io/v1","spec":{"audiences":["%s"]}}
`, aud)))
	if err != nil {
		return "", err
	}
	var secret CreateTokenResponse
	err = json.Unmarshal(data, &secret)
	if err != nil {
		return "", err
	}

	return secret.Status.Token, nil
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
	}
}

func (uk *UK8S) initDefaultTokenSource(ctx context.Context) error {
	// Init GCP auth
	// DefaultTokenSource will:
	// - check GOOGLE_APPLICATION_CREDENTIALS
	// - ~/.config/gcloud/application_default_credentials.json"
	// - use metadata
	ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return err
	}
	uk.TokenProvider = func(ctx context.Context, s string) (string, error) {
		t, err := ts.Token()
		if err != nil {
			return "", err
		}
		return t.AccessToken, nil
	}
	return nil
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
	m.TokenProvider = uk
	uk.Client = uk.httpClient(nil)

	err := uk.initDefaultTokenSource(ctx)
	if err != nil {
		return nil, err
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

			def, cbyl, err := extractClusters(uk, kc)
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

		_, err = getGKEClusters(ctx, uk, accessToken, m.ProjectId)
		if err != nil {
			return nil, err
		}
	}

	// TODO: select the config clusters
	// TODO: function to update the 'active' cluster on failure, locate local one

	return uk, nil
}
