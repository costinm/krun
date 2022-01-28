package main

import (
	"context"
	"crypto/x509"
	"flag"
	"log"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/cloud-run-mesh/pkg/gcp"
	"github.com/GoogleCloudPlatform/cloud-run-mesh/pkg/mesh"
	"github.com/GoogleCloudPlatform/cloud-run-mesh/pkg/sts"
	"google.golang.org/grpc"

	"github.com/costinm/hbone"
	"github.com/costinm/krun/third_party/istio/cas"
	"github.com/costinm/krun/third_party/istio/istioca"
	"github.com/costinm/krun/third_party/istio/meshca"
)

var (
	ns       = flag.String("n", "fortio", "Namespace")
	aud      = flag.String("audience", "", "Audience to use in the CSR request")
	provider = flag.String("addr", "", "Address. If empty will use the cluster default. meshca or cas can be used as shortcut")
)

// CLI to get the mesh certificates, using MeshCA, CAS os Istio CA.
func main() {
	flag.Parse()
	startCtx := context.Background()

	kr := mesh.New()
	if *ns != "" {
		kr.Namespace = *ns
	}
	ctx := context.Background()

	// Using the micro or real k8s client.
	if false {
		//_, err := urest.K8SClient(startCtx, kr)
		//err = kr.LoadConfig(startCtx)
		//if err != nil {
		//	panic(err)
		//}
	} else {
		err := gcp.InitGCP(ctx, kr)
		if err != nil {
			log.Fatal("Failed to find K8S ", time.Since(kr.StartTime), kr, os.Environ(), err)
		}
		err = kr.LoadConfig(context.Background())
		if err != nil {
			log.Fatal("Failed to connect to mesh ", time.Since(kr.StartTime), kr, os.Environ(), err)
		}

		//k8s := &k8s.K8S{Mesh: kr}
		//k8s.VendorInit = gcp.InitGCP
		//kr.Cfg = k8s
		//kr.TokenProvider = k8s

		// Init K8S client, using official API server.
		// Will attempt to use GCP API to load metadata and populate the fields
		//k8s.K8SClient(startCtx)

		// Load mesh-env and other configs from k8s.

	}

	// Need the settings from mesh-env
	f, err := initOTel(startCtx, kr)
	if err != nil {
		log.Println("OTel init failed", err)
	} else {
		defer f()
	}

	if kr.MeshConnectorAddr == "" {
		log.Fatal("Failed to find in-cluster, missing 'hgate' service in mesh env")
	}

	kr.XDSAddr = kr.MeshConnectorAddr + ":15012"

	// Used to generate the CSR
	auth := hbone.NewAuth()
	priv, csr, err := auth.NewCSR("rsa", kr.TrustDomain, "spiffe://"+kr.TrustDomain+"/ns/"+kr.Namespace+"/sa/"+kr.KSA)
	if err != nil {
		log.Fatal("Failed to find mesh certificates ", err)
	}

	// TODO: fetch public keys too - possibly from all

	if *provider == "meshca" {
		InitMeshCA(kr, auth, csr, priv)
	} else if *provider == "cas" {
		InitCAS(kr, auth, csr, priv)
	} else {
		InitMeshCert(kr, auth, csr, priv)
	}
	cert, err := x509.ParseCertificate(auth.Cert.Certificate[0])
	if err != nil {
		panic(err)
	}

	log.Println(cert.URIs, cert.Subject)
	time.Sleep(4 * time.Second)
}

func InitMeshCert(kr *mesh.KRun, auth *hbone.Auth, csr []byte, priv []byte) {
	if kr.CitadelRoot != "" && kr.MeshConnectorAddr != "" {
		auth.AddRoots([]byte(kr.CitadelRoot))

		grpccreds, _ := sts.NewSTS(kr)
		// Audience: "istio-ca"
		cca, err := istioca.NewCitadelClient(&istioca.Options{
			TokenProvider: grpccreds,
			CAEndpoint:    kr.MeshConnectorAddr + ":15012",
			TrustedRoots:  auth.TrustedCertPool,
			CAEndpointSAN: "istiod.istio-system.svc",
			GRPCOptions:   OTELGRPCClient(),
		})
		if err != nil {
			log.Fatal(err)
		}
		chain, err := cca.CSRSign(csr, 24*3600)

		//log.Println(chain, err)
		err = auth.SetKeysPEM(priv, chain)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		InitMeshCA(kr, auth, csr, priv)
	}
}

func InitMeshCA(kr *mesh.KRun, auth *hbone.Auth, csr []byte, priv []byte) {
	tokenProvider, err := sts.NewSTS(kr)
	tokenProvider.UseAccessToken = true // even if audience is provided.

	// TODO: Use MeshCA if citadel is not in cluster
	var ol []grpc.DialOption
	ol = append(ol, grpc.WithPerRPCCredentials(tokenProvider))
	ol = append(ol, OTELGRPCClient()...)

	mca, err := meshca.NewGoogleCAClient("meshca.googleapis.com:443",
		ol)
	chain, err := mca.CSRSign(csr, 24*3600)

	if err != nil {
		log.Fatal(err)
	}
	//log.Println(chain, priv)
	err = auth.SetKeysPEM(priv, chain)
	if err != nil {
		log.Fatal(err)
	}

}

// TODO
func InitCAS(kr *mesh.KRun, auth *hbone.Auth, csr []byte, priv []byte) {
	// TODO: Use MeshCA if citadel is not in cluster
	tokenProvider, err := sts.NewSTS(kr)

	// This doesn't work
	//  Could not use the REFLECTED_SPIFFE subject mode because the caller does not have a SPIFFE identity. Please visit the CA Service documentation to ensure that this is a supported use-case
	//  tokenProvider.MDPSA = true
	// The token MUST be the federated access token

	tokenProvider.UseAccessToken = true // even if audience is provided.

	var ol []grpc.DialOption
	ol = append(ol, grpc.WithPerRPCCredentials(tokenProvider))
	ol = append(ol, OTELGRPCClient()...)

	mca, err := cas.NewGoogleCASClient("projects/"+kr.ProjectId+
		"/locations/"+kr.Region()+"/caPools/istio", ol)
	chain, err := mca.CSRSign(csr, 24*3600)

	if err != nil {
		log.Fatal(err)
	}
	//log.Println(chain, priv)
	err = auth.SetKeysPEM(priv, chain)
	if err != nil {
		log.Fatal(err)
	}

}
