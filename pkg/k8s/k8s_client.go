package k8s

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Init klog.InitFlags from an env (to avoid messing with the CLI of the app)
func init() {
	fs := &flag.FlagSet{}
	kf := strings.Split(os.Getenv("KLOG_FLAGS"), " ")
	fs.Parse(kf)
	klog.InitFlags(fs)
}

func RegionFromMetadata() (string, error) {
	v, err := queryMetadata("http://metadata.google.internal/computeMetadata/v1/instance/region")
	if err != nil {
		return "", err
	}
	vs := strings.SplitAfter(v, "/regions/")
	if len(vs) != 2 {
		return "", fmt.Errorf("malformed region value split into %#v", vs)
	}
	return vs[1], nil
}

func ProjectFromMetadata() (string, error) {
	v, err := queryMetadata("http://metadata.google.internal/computeMetadata/v1/project/project-id")
	if err != nil {
		return "", err
	}
	return v, nil
}

func ProjectNumberFromMetadata() (string, error) {
	v, err := queryMetadata("http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id")
	if err != nil {
		return "", err
	}
	return v, nil
}

func queryMetadata(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata server responeded with code=%d %s", resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), err
}

// InitUsingKubeConfig uses in-cluster or KUBECONFIG to init the primary k8s cluster.
func (kr *KRun) InitUsingKubeConfig() error {
	// Explicit kube config - use it
	kc := os.Getenv("KUBECONFIG")
	if kc == "" {
		kc = os.Getenv("HOME") + "/.kube/config"
	}
	if _, err := os.Stat(kc); err == nil {
		config, err := clientcmd.BuildConfigFromFlags("", kc)
		if err != nil {
			return err
		}
		kr.Client, err = kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (kr *KRun) InitInCluster() error {
	// In cluster
	hostInClustser := os.Getenv("KUBERNETES_SERVICE_HOST")
	if hostInClustser != "" {
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
		kr.Client, err = kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (kr *KRun) InitGCP() error {
	// Get all clusters, including hub - attempt to find the cluster.
	//kr.AllHub(gcpProj, cluster, "", "")
	t0 := time.Now()
	if kr.ClusterName == "" {
		// WIP - list all clusters and attempt to find one in the same region.
		// TODO: connect to cluster, find istiod - and keep trying until a working
		// one is found ( fallback )

		// ~500ms
		kr.AllClusters(kr.ProjectId, "", "", "")
		//log.Println("Get all clusters ", time.Since(t0))

		for _, c := range kr.Clusters {
			if strings.HasPrefix(c.Location, kr.ClusterLocation) {
				log.Println("------- Found ", c)
				rc, err := kr.restConfigForCluster(c)
				if err != nil {
					continue
				}
				kr.Client, err = kubernetes.NewForConfig(rc)
				if err != nil {
					continue
				}
				break
			}
		}
	} else {
		// ~400 ms
		rc, err := kr.CreateRestConfig(kr.ProjectId, kr.ClusterLocation, kr.ClusterName)
		if err != nil {
			return err
		}
		kr.Client, err = kubernetes.NewForConfig(rc)
		if err != nil {
			return err
		}
		log.Println("Get 1 cluster ", time.Since(t0))
	}

	kr.SaveKubeConfig()

	return nil
}

// GetK8S gets the default k8s client, using environment variables to decide how.
//
func (kr *KRun) GetK8S() (*kubernetes.Clientset, error) {
	if kr.Client != nil {
		return kr.Client, nil
	}

	err := kr.InitUsingKubeConfig()
	if err != nil {
		return nil, err
	}
	if kr.Client != nil {
		return kr.Client, nil
	}

	err = kr.InitGCP()
	if err != nil {
		return nil, err
	}
	if kr.Client != nil {
		return kr.Client, nil
	}

	return nil, errors.New("not found")
}
