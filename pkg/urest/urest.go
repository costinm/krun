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
	"runtime"
	"sync"
	"time"

	"golang.org/x/oauth2/google"
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

var Debug = false

var (
	projectName    = "konfig"
	projectVersion = "0.1.0"
	projectURL     = "https://github.com/costinm/konfig"
	userAgent      = fmt.Sprintf("%s/%s (+%s; %s)",
		projectName, projectVersion, projectURL, runtime.Version())
)

// MeshSettings has common settings for all clients
type MeshSettings struct {
	// If TransportWrapper is set, the http clients will be wrapped
	// This is intended for integration with OpenTelemetry or other transport wrappers.
	TransportWrapper func(transport http.RoundTripper) http.RoundTripper

	// Hub or user project ID. If set, will be used to lookup clusters.
	ProjectId      string
	Namespace      string
	ServiceAccount string

	// Location where the workload is running, to select local clusters.
	Location string
}

// URest is a micro REST client, modeled after k8s and GCP REST model, using only base http and a token source.
// It can work with most REST APIs - but client is expected to know how to generate the raw payload and
// parse the response.
type URest struct {

	// Client using system certificates, for external sites.
	// The URestClient has a separate Client, configured to authenticate servers using a custom root CA.
	Client *http.Client

	// TokenProvider is the default source of tokens for this client.
	// It is expected to return tokens with the given audience. Google OAuth2 provider for example
	// works using 'default app credentials' or metadata server.
	TokenProvider func(context.Context, string) (string, error)

	// Configs and defaults for the client. It is assumed the client is using a project ID, and has k8s or equivalent
	// 'namespace', 'service account' and location. This may also be populated automatically.
	Mesh *MeshSettings

	// Default cluster - each cluster has separate credentials and TLS certs.
	// TODO: find better name, this can be any REST target that uses custom certs.
	Current *URestClient

	// Clients by name
	Clusters map[string]*URestClient

	// Clients by location
	ClustersByLocation map[string][]*URestClient

	m sync.RWMutex
}

// URestClient is a wrapper around a http client configured with specific certs and auth provider.
// Based on K8S Cluster config.
type URestClient struct {
	URest *URest
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
	Client *http.Client
}

func (uK8S *URest) String() string {
	return uK8S.Current.Id
}

// HttpClient returns a http.Client configured with the specified root CA, and reasonable settings.
// The URest wrapper is added, for telemetry or other interceptors.
func (uK8S *URest) HttpClient(caCert []byte) *http.Client {
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
	if uK8S.Mesh.TransportWrapper != nil {
		rt = uK8S.Mesh.TransportWrapper(rt)
	}

	return &http.Client{
		Transport: rt,
	}

}

func (uk8s *URest) DoNsKindName(ctx context.Context, ns, kind, name string, postdata []byte) ([]byte, error) {
	resourceURL := uk8s.Current.Base + fmt.Sprintf("/api/v1/namespaces/%s/%ss/%s",
		ns, kind, name)
	return uk8s.Do(ctx, resourceURL, postdata)
}

func (uk8s *URest) Do(ctx context.Context, resourceURL string, postdata []byte) ([]byte, error) {

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
		return nil, errors.New(fmt.Sprintf("kconfig: unable to get %s, status code %v",
			resourceURL, resp.StatusCode))
	}

	return data, nil
}

func New() *URest {
	return &URest{
		ClustersByLocation: map[string][]*URestClient{},
		Clusters:           map[string]*URestClient{},
		Client:             http.DefaultClient,
	}
}

func (uk *URest) InitDefaultTokenSource(ctx context.Context) error {
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
