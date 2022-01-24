package urest

import (
	"context"
	"encoding/json"
	"fmt"
)

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
