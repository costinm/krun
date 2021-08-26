package k8s

import (
	"context"
	"log"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CheckServices will look for istiod, hgate and the debug service.
//
// TODO: detect istiod service (in-cluster), use it if external is not configured
// TODO: detect cert-ssh, use it to enable debug
func (kr *KRun) CheckServices(ctx context.Context, client *kubernetes.Clientset) error {
	ts, err := client.CoreV1().Services("istio-system").List(ctx,
		metav1.ListOptions{})
	if err != nil {
		log.Println("Error listing ", err)
		return err
	}

	for _, s := range ts.Items {
		if s.Name == "cert-ssh" {
			log.Println("Found cert-ssh", s.Status)
		}
		if strings.HasPrefix(s.Name, "istiod") {
			log.Println("Found istiod", s.Name, s.Status)
		}
	}
	return nil
}

// ConnectHGate will connect to an in-cluster reverse gateway, and maintain the connection.
//
func (kr *KRun) FindHGate(ctx context.Context) (string, error) {

	ts, err := kr.Client.CoreV1().Services("hgate").Get(ctx, "hgate", metav1.GetOptions{})
	if err != nil {
		log.Println("Error getting service hgate ", err)
		return "", err
	}

	if len(ts.Status.LoadBalancer.Ingress) > 0 {
		return ts.Status.LoadBalancer.Ingress[0].IP, nil
	}

	//te, err := client.CoreV1().Endpoints("hgate").Get(ctx, "hgate", metav1.GetOptions{})
	//if err != nil {
	//	log.Println("Error listing ", err)
	//	return err
	//}
	//
	//for _, s := range ts.Items {
	//	if s.Name == "hgate" {
	//		log.Println("Found hgate", s.Status)
	//	}
	//	if s.Name == "cert-ssh" {
	//		log.Println("Found cert-ssh", s.Status)
	//	}
	//	if strings.HasPrefix(s.Name, "istiod") {
	//		log.Println("Found istiod",  s.Name, s.Status)
	//	}
	//}
	return "", nil
}
