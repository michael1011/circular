package main

import (
	"circular/singleton"
	"github.com/elementsproject/glightning/glightning"
)

func registerHooks(p *glightning.Plugin) {
	p.RegisterHooks(&glightning.Hooks{
		HtlcAccepted: OnHtlcAccepted,
	})
}

func OnHtlcAccepted(event *glightning.HtlcAcceptedEvent) (*glightning.HtlcAcceptedResponse, error) {
	node := singleton.GetNode()
	preimage, err := node.GetFromDb(event.Htlc.PaymentHash)
	if err != nil {
		return event.Continue(), nil
	}
	node.Logln(glightning.Info, "resolving HTLC with preimage: ", string(preimage))
	return event.Resolve(string(preimage)), nil
}
