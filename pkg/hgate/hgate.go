package hgate

import (
	"context"
	"fmt"
	"log"

	"github.com/costinm/hbone"
)

type HGate struct {
	auth *hbone.Auth
	hb *hbone.HBone
}

func New(hg string, name, namespace string) *HGate {
	auth, err := hbone.LoadAuth("")
	if err != nil {
		log.Fatal("Failed to find mesh certificates", err)
	}

	hb := hbone.New(auth)

	_, err = hbone.ListenAndServeTCP(":15009", hb.HandleAcceptedH2C)
	if err != nil {
		log.Fatal("Failed to start h2c on 15009", err)
	}

	if  hg == "" {
		log.Println("hgate not found, not attaching to the cluster", err)
	} else {
		attachC := hb.NewClient(name + "." + namespace + ":15009")
		attachE := attachC.NewEndpoint("")
		attachE.SNI = fmt.Sprintf("outbound_.8080_._.%s.%s.svc.cluster.local", name, namespace)
		go func() {
			_, err := attachE.DialH2R(context.Background(), hg+":15441")
			log.Println("H2R connected", hg, err)
		}()
	}
	return &HGate{auth: auth, hb: hb}
}
