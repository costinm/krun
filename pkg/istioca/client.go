// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative  ca.proto

package istioca

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

)

const (
	// CertSigner info
	CertSigner = "CertSigner"
)

type Options struct {
	CAEndpoint string
	CAEndpointSAN string
	TokenProvider      credentials.PerRPCCredentials
	CertSigner string
	ClusterID string
	ProvCert string
}

type CitadelClient struct {
	enableTLS     bool
	caTLSRootCert []byte
	client        IstioCertificateServiceClient
	conn          *grpc.ClientConn
	opts          *Options
	usingMtls     bool
}

// NewCitadelClient create a CA client for Citadel.
func NewCitadelClient(opts *Options, tls bool, rootCert []byte) (*CitadelClient, error) {
	c := &CitadelClient{
		enableTLS:     tls,
		caTLSRootCert: rootCert,
		opts:          opts,
		usingMtls:     false,
	}

	conn, err := c.buildConnection()
	if err != nil {
		log.Printf("Failed to connect to endpoint %s: %v", opts.CAEndpoint, err)
		return nil, fmt.Errorf("failed to connect to endpoint %s", opts.CAEndpoint)
	}
	c.conn = conn
	c.client = NewIstioCertificateServiceClient(conn)
	return c, nil
}

func (c *CitadelClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// CSR Sign calls Citadel to sign a CSR.
func (c *CitadelClient) CSRSign(csrPEM []byte, certValidTTLInSec int64) ([]string, error) {
	crMetaStruct := &Struct{
		Fields: map[string]*Value{
			CertSigner: {
				Kind: &Value_StringValue{StringValue: c.opts.CertSigner},
			},
		},
	}
	req := &IstioCertificateRequest{
		Csr:              string(csrPEM),
		ValidityDuration: certValidTTLInSec,
		Metadata:         crMetaStruct,
	}
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("ClusterID", c.opts.ClusterID))
	resp, err := c.client.CreateCertificate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %v", err)
	}

	if len(resp.CertChain) <= 1 {
		return nil, errors.New("invalid empty CertChain")
	}

	return resp.CertChain, nil
}

func (c *CitadelClient) getTLSDialOption() (grpc.DialOption, error) {
	// Load the TLS root certificate from the specified file.
	// Create a certificate pool
	var certPool *x509.CertPool
	var err error
	if c.caTLSRootCert == nil {
		// No explicit certificate - assume the citadel-compatible server uses a public cert
		certPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
	} else {
		certPool = x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(c.caTLSRootCert)
		if !ok {
			return nil, fmt.Errorf("failed to append certificates")
		}
	}
	var certificate tls.Certificate
	config := tls.Config{
		Certificates: []tls.Certificate{certificate},
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			if c.opts.ProvCert != "" {
				// Load the certificate from disk
				certificate, err = tls.LoadX509KeyPair(
					filepath.Join(c.opts.ProvCert, "cert-chain.pem"),
					filepath.Join(c.opts.ProvCert, "key.pem"))

				if err != nil {
					// we will return an empty cert so that when user sets the Prov cert path
					// but not have such cert in the file path we use the token to provide verification
					// instead of just broken the workflow
					log.Printf("cannot load key pair, using token instead: %v", err)
					return &certificate, nil
				}
				var isExpired bool
				isExpired, err = c.isCertExpired(filepath.Join(c.opts.ProvCert, "cert-chain.pem"))
				if err != nil {
					log.Printf("cannot parse the cert chain, using token instead: %v", err)
					return &tls.Certificate{}, nil
				}
				if isExpired {
					log.Printf("cert expired, using token instead")
					return &tls.Certificate{}, nil
				}
				c.usingMtls = true
			}
			return &certificate, nil
		},
	}
	config.RootCAs = certPool

	// For debugging on localhost (with port forward)
	// TODO: remove once istiod is stable and we have a way to validate JWTs locally
	if strings.Contains(c.opts.CAEndpoint, "localhost") {
		config.ServerName = "istiod.istio-system.svc"
	}

	transportCreds := credentials.NewTLS(&config)
	return grpc.WithTransportCredentials(transportCreds), nil
}

func (c *CitadelClient) isCertExpired(filepath string) (bool, error) {
	var err error
	var certPEMBlock []byte
	certPEMBlock, err = os.ReadFile(filepath)
	if err != nil {
		return true, fmt.Errorf("failed to read the cert, error is %v", err)
	}
	var certDERBlock *pem.Block
	certDERBlock, _ = pem.Decode(certPEMBlock)
	if certDERBlock == nil {
		return true, fmt.Errorf("failed to decode certificate")
	}
	x509Cert, err := x509.ParseCertificate(certDERBlock.Bytes)
	if err != nil {
		return true, fmt.Errorf("failed to parse the cert, err is %v", err)
	}
	return x509Cert.NotAfter.Before(time.Now()), nil
}

func (c *CitadelClient) buildConnection() (*grpc.ClientConn, error) {
	var opts grpc.DialOption
	var err error
	if c.enableTLS {
		opts, err = c.getTLSDialOption()
		if err != nil {
			return nil, err
		}
	} else {
		opts = grpc.WithInsecure()
	}

	conn, err := grpc.Dial(c.opts.CAEndpoint,
		opts,
		grpc.WithPerRPCCredentials(c.opts.TokenProvider))
		//security.CARetryInterceptor())
	if err != nil {
		log.Println("Failed to connect to endpoint %s: %v", c.opts.CAEndpoint, err)
		return nil, fmt.Errorf("failed to connect to endpoint %s", c.opts.CAEndpoint)
	}

	return conn, nil
}

// GetRootCertBundle: Citadel (Istiod) CA doesn't publish any endpoint to retrieve CA certs
func (c *CitadelClient) GetRootCertBundle() ([]string, error) {
	return []string{}, nil
}