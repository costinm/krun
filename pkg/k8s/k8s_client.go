package k8s

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

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

// initUsingKubeConfig uses KUBECONFIG or $HOME/.kube/config
// to init the primary k8s cluster.
//
// error is set if KUBECONFIG is set or ~/.kube/config exists and
// fail to load. If the file doesn't exist, err is nil.
func (kr *KRun) initUsingKubeConfig() error {
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

func (kr *KRun) initInCluster() error {
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


// WIP
func New() (*KRun, error) {
	kr := &KRun{}

	return kr, nil
}

// InitK8SClient gets the default k8s client, using environment
// variables to decide how:
//
// - KUBECONFIG or $HOME/.kube/config will be tried first
// - GKE is checked - using metadata server to get
//   PROJECT_ID, CLUSTER_LOCATION (if not set).
// - (in future other vendor-specific methods may be added)
// - finally in-cluster will be checked.
//
// Once the cluster is found, additional config can be loaded from
// the cluster.
func (kr *KRun) InitK8SClient() error {
	if kr.Client != nil {
		return  nil
	}

	err := kr.initUsingKubeConfig()
	if err != nil {
		return  err
	}
	if kr.Client != nil {
		return  nil
	}

	err = kr.initGCP()
	if err != nil {
		return  err
	}
	if kr.Client != nil {
		return  nil
	}

	err = kr.initInCluster()
	if err != nil {
		return  err
	}
	if kr.Client != nil {
		return  nil
	}

	return errors.New("not found")
}
