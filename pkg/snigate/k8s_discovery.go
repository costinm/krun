package snigate

import (
	"context"
	"log"

	"github.com/costinm/krun/pkg/k8s"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
)

// On client connection, create a WorkloadEntry or EndpointSlice so
// Istio is able to connect using the gateway IP and SNI port.

// Implementation notes:
// For WorkloadEntry, Istio name is based on group-ip0-network, truncated to 253
// (workloadentry_controller.go), using AUTO_REGISTER_GROUP meta.
//
//


func UpdateSlice(ctx context.Context, kr *k8s.KRun, ns string,
	name string) {
	es := &discoveryv1.EndpointSlice{}
	kr.Client.DiscoveryV1().EndpointSlices(ns).Get(
		ctx, name, metav1.GetOptions{})
	kr.Client.DiscoveryV1().EndpointSlices(ns).Create(
		ctx, es, metav1.CreateOptions{})
	kr.Client.DiscoveryV1().EndpointSlices(ns).Update(
		ctx, es, metav1.UpdateOptions{})
}

// NewSliceWatcher keeps track of endpoint slices.
// Currently for debugging/dev - long term we may re-forward
// if the reverse tunnel moves to a new instance.
func NewSliceWatcher(kr *k8s.KRun) {
	inF := informers.NewSharedInformerFactory(kr.Client, 0)
	stop := make(chan struct{})
	inF.Start(stop)
	esi := inF.Discovery().V1beta1().EndpointSlices().Informer()
	es := &EndpointSlices{}
	esi.AddEventHandler(es)
	esi.Run(stop)

}

type EndpointSlices struct {}

func (e EndpointSlices) OnAdd(obj interface{}) {
	log.Println("Add", obj)
}

func (e EndpointSlices) OnUpdate(oldObj, newObj interface{}) {
	log.Println("Update", newObj)
}

func (e EndpointSlices) OnDelete(obj interface{}) {
	log.Println("Del", obj)
}

