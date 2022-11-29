package singleton

import (
	"circular/graph"
	"github.com/elementsproject/glightning/glightning"
	"sync"
)

type Node interface {
	Stopped() bool
	SetStopped(stopped bool)

	GetPeersLock() *sync.RWMutex

	GetId() string
	GetGraph() *graph.Graph
	GetPeers() map[string]*glightning.Peer
	GetPeerChannelFromGraphChannel(graphChannel *graph.Channel) (*glightning.PeerChannel, error)
	GetBestPeerChannel(id string, metric func(*glightning.PeerChannel) uint64) *glightning.PeerChannel
	GetOutgoingChannelFromScid(scid string) (*graph.Channel, error)
	GetIncomingChannelFromScid(scid string) (*graph.Channel, error)
	GetGraphChannelFromPeerChannel(channel *glightning.PeerChannel, direction string) (*graph.Channel, error)

	IsPeerConnected(channel *glightning.PeerChannel) bool

	GeneratePreimageHashPair() (string, error)
	UpdateChannelBalance(outPeer, inPeer, outScid, inScid string, amount uint64)

	SaveToDb(key string, value any) error
	GetFromDb(key string) ([]byte, error)

	SendRebalanceAttempt(route *graph.PrettyRoute)

	SendPay(route *graph.Route, paymentHash string) (*glightning.SendPayFields, error)

	OnPaymentFailure(sf *glightning.SendPayFailure)
	OnPaymentSuccess(ss *glightning.SendPaySuccess)
	OnConnect(c *glightning.ConnectEvent)
	OnDisconnect(c *glightning.DisconnectEvent)

	Logln(level glightning.LogLevel, v ...any)
	Logf(level glightning.LogLevel, format string, v ...any)
}

var (
	singleton Node
)

func SetNode(n Node) {
	singleton = n
}

func GetNode() Node {
	return singleton
}
