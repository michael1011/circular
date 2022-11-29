package main

import (
	"circular/singleton"
	"github.com/elementsproject/glightning/glightning"
)

// TODO: listen for `channel_state_changed` and `channel_opened`
// 		so we don't have to refresh peer list every time
// TODO: listen for `shutdown`

func registerSubscriptions(p *glightning.Plugin) {
	p.SubscribeSendPayFailure(OnSendPayFailure)
	p.SubscribeSendPaySuccess(OnSendPaySuccess)
	p.SubscribeConnect(OnConnect)
	p.SubscribeDisconnect(OnDisconnect)
}

func OnSendPayFailure(sf *glightning.SendPayFailure) {
	singleton.GetNode().OnPaymentFailure(sf)
}

func OnSendPaySuccess(ss *glightning.SendPaySuccess) {
	singleton.GetNode().OnPaymentSuccess(ss)
}

func OnConnect(c *glightning.ConnectEvent) {
	singleton.GetNode().OnConnect(c)
}

func OnDisconnect(d *glightning.DisconnectEvent) {
	singleton.GetNode().OnDisconnect(d)
}
