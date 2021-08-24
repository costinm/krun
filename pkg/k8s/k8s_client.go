package k8s

import (
	"context"
	"errors"
	"flag"
	"log"
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
		cf, err := clientcmd.LoadFromFile(kc)
		//config := clientcmd.NewNonInteractiveClientConfig(cf, cf.CurrentContext, nil, nil)
		if strings.HasPrefix(cf.CurrentContext, "gke_") {
			parts := strings.Split(cf.CurrentContext, "_")
			if len(parts) > 3 {
				// TODO: if env variable with cluster name/location are set - use that for context
				kr.ProjectId = parts[1]
				kr.ClusterLocation = parts[2]
				kr.ClusterName = parts[3]
			}
		}
		if strings.HasPrefix(cf.CurrentContext, "connectgateway_") {
			parts := strings.Split(cf.CurrentContext, "_")
			if len(parts) > 2 {
				// TODO: if env variable with cluster name/location are set - use that for context
				kr.ProjectId = parts[1]
				kr.ClusterName = parts[2]
			}
		}

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
	if kr.Client != nil {
		return nil
	}
	hostInClustser := os.Getenv("KUBERNETES_SERVICE_HOST")
	if hostInClustser != "" {
		log.Println("Using in-cluster config: ", hostInClustser)
		kr.InCluster = true
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
func (kr *KRun) InitK8SClient(ctx context.Context) error {
	if kr.Client != nil {
		return  nil
	}

	err := kr.initUsingKubeConfig()
	if err != nil {
		return  err
	}

	err = kr.initInCluster()
	if err != nil {
		return  err
	}

	if kr.VendorInit != nil {
		err = kr.VendorInit(ctx, kr)
		if err != nil {
			return  err
		}
	}
	if kr.Client != nil {
		return  nil
	}

	return errors.New("not found")
}
