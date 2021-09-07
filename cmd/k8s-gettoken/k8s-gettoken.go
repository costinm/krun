//Copyright 2021 Google LLC
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    https://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"

	_ "github.com/costinm/cloud-run-mesh/pkg/gcp"
	k8s "github.com/costinm/cloud-run-mesh/pkg/mesh"
)

var (
	nsFlag  = flag.String("ns", "default", "namespace")
	ksaFlag = flag.String("ksa", "default", "kubernetes service account")
)

// Minimal tool to get a K8S token with audience.
func main() {
	flag.Parse()
	aud := "api"
	if len(flag.Args()) > 1 {
		aud = flag.Args()[0]
	}

	kr := k8s.New("")
	if kr.Namespace == "" {
		kr.Namespace = *nsFlag
	}
	if kr.KSA == "" {
		kr.KSA = *ksaFlag
	}
	err := kr.LoadConfig(context.Background())
	if err != nil {
		panic(err)
	}

	tok, err := kr.GetToken(context.Background(), aud)
	if err != nil {
		panic(err)
	}

	fmt.Println(tok)
}
