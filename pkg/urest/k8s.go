package urest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// Copyright 2019 The Konfig Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

// Uses URestClient to perform raw K8S operations with minimal depenencies.
//
type UK8S struct {
	c *URestClient
}

// Initializes a K8S client, based on kube config, GKE or hub.
// Will pick 'default' cluster from kube config, or a local config cluster from GKE.
// The other clusters will also be loaded and will be available.
func NewUK8S(ctx context.Context, ur *URest, hub, project, location, name string) (*UK8S, error) {
	return nil, nil
}

// Init the K8S client:
//
// 1. Explicit KUBECONFIG
// 2. GCP APIs, selecting a cluster.
//
func K8SClient(ctx context.Context, m *MeshSettings) (*URest, error) {
	uk := New()
	uk.Mesh = m
	//m.Cfg = uk
	//m.TokenProvider = uk
	uk.Client = http.DefaultClient
	//uk.Client = uk.HttpClient(nil)

	err := uk.InitDefaultTokenSource(ctx)
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

func (uK8S *URest) GetSecret(ctx context.Context, ns string, name string) (map[string][]byte, error) {
	data, err := uK8S.DoNsKindName(ctx, ns, "secret", name, nil)
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

func (uK8S *URest) GetCM(ctx context.Context, ns string, name string) (map[string]string, error) {
	data, err := uK8S.DoNsKindName(ctx, ns, "configmap", name, nil)
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

func (uK8S *URest) GetToken(ctx context.Context, aud string) (string, error) {
	return uK8S.GetTokenRaw(ctx, uK8S.Mesh.Namespace, uK8S.Mesh.ServiceAccount, aud)
}

func (uK8S *URest) GetTokenRaw(ctx context.Context, ns, name, aud string) (string, error) {
	// If no audience is specified, something like
	//   https://container.googleapis.com/v1/projects/costin-asm1/locations/us-central1-c/clusters/big1
	// is generated ( on GKE ) - which seems to be the audience for K8S
	data, err := uK8S.DoNsKindName(ctx, ns, "serviceaccount", name+"/token", []byte(fmt.Sprintf(`
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

// RequireTranportSecurity is part of gRPC interface, returning false because we also support secure networks (low-level)
func (istiodTP *URestClient) RequireTransportSecurity() bool {
	return false
}

// GetRequestMetadata implements credentials.PerRPCCredentials, specifically for 'trustDomain' tokens used by
// Istiod. Audience example: https://istiod.istio-system.svc/istio.v1.auth.IstioCertificateService (based on SNI name!)
func (istiodTP *URestClient) GetRequestMetadata(ctx context.Context, aud ...string) (map[string]string, error) {
	a := aud[0]
	if len(aud) > 0 && strings.Contains(aud[0], "/istio.v1.auth.IstioCertificateService") {
		a = "istio-ca"
		//a = istiodTP.URest.Mesh.TrustDomain
	}
	//if istiodTP.Audience != "" {
	//	a = istiodTP.Audience // override
	//}
	// TODO: same for the XDS stream

	kt, err := istiodTP.URest.GetToken(ctx, a)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"authorization": "Bearer " + kt,
	}, nil
}
