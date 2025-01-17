package node

import (
	"circular/graph"
	"circular/singleton"
	"circular/util"
	"fmt"
	"github.com/elementsproject/glightning/glightning"
	"golang.org/x/net/websocket"
	"log"
	"math/rand"
	"sync"
	"time"
)

const (
	CIRCULAR_DIR                     = "circular"
	DEFAULT_PEER_REFRESH_INTERVAL    = 30  // seconds
	DEFAULT_LIQUIDITY_RESET_INTERVAL = 300 // minutes
	DEFAULT_RPC_TIMEOUT              = 60  // seconds

	OptionWebSocketEndpoint = "circular-websocket"

	DefaultWebSocketEndpoint = "0.0.0.0:8222"
)

type Node struct {
	lightning           *glightning.Lightning
	plugin              *glightning.Plugin
	liquidityRefresh    time.Duration
	initLock            *sync.Mutex
	saveStats           bool
	PeersLock           *sync.RWMutex
	Id                  string
	Peers               map[string]*glightning.Peer
	Graph               *graph.Graph
	DB                  *Store
	LiquidityUpdateChan chan *LiquidityUpdate

	stopped bool

	activeWebSockets     []*websocket.Conn
	activeWebSocketsLock sync.Mutex
}

func NewNode() *Node {
	rand.Seed(time.Now().UnixNano())
	node := &Node{
		initLock:            &sync.Mutex{},
		PeersLock:           &sync.RWMutex{},
		Peers:               make(map[string]*glightning.Peer),
		LiquidityUpdateChan: make(chan *LiquidityUpdate, 16),
	}
	singleton.SetNode(node)
	return node
}

func (n *Node) Init(lightning *glightning.Lightning, plugin *glightning.Plugin, options map[string]glightning.Option, config *glightning.Config) {
	defer util.TimeTrack(time.Now(), "node.Init", n.Logf)
	n.initLock.Lock()
	defer n.initLock.Unlock()

	n.setOptions(lightning, plugin, options)

	n.Logln(glightning.Debug, "getting ID")
	info, err := n.lightning.GetInfo()
	if err != nil {
		log.Fatalln("GetInfo failed in init, exiting")
	}
	n.Id = info.Id

	n.Logln(glightning.Debug, "loading from file")
	n.getGraphFromFile(err, config)

	n.Logln(glightning.Debug, "refreshing graph")
	if err = n.refreshGraph(); err != nil {
		log.Fatalln("RefreshGraph failed in init, exiting")
	}

	n.Logln(glightning.Debug, "refreshing peers")
	if err = n.refreshPeers(); err != nil {
		log.Fatalln("RefreshPeers failed in init, exiting")
	}

	n.Logln(glightning.Debug, "opening database")
	n.DB = NewDB(config.LightningDir + "/" + CIRCULAR_DIR)

	n.Logln(glightning.Debug, "setting up cronjobs")
	n.setupCronJobs(options)

	n.startWebsocket(options)

	n.Logln(glightning.Info, "node initialized")
}

func (n *Node) getGraphFromFile(err error, config *glightning.Config) {
	err = n.LoadGraphFromFile(config.LightningDir+"/"+CIRCULAR_DIR, graph.FILE)
	if err == util.ErrNoGraphToLoad {
		// If we don't have a graph, we need to create one
		n.Logln(glightning.Unusual, err)
		n.Graph = graph.NewGraph()
	} else if err != nil {
		// If we have an error, we're in trouble
		n.Logln(glightning.Unusual, err)
		log.Fatalln(err)
	}
}

func (n *Node) setOptions(lightning *glightning.Lightning, plugin *glightning.Plugin, options map[string]glightning.Option) {
	n.lightning = lightning
	n.plugin = plugin
	n.Logln(glightning.Info, "initializing node")

	n.liquidityRefresh = time.Duration(options["circular-liquidity-refresh"].GetValue().(int)) * time.Minute
	n.Logln(glightning.Debug, "liquidity refresh interval: ", int(n.liquidityRefresh.Minutes()), " minutes")

	n.saveStats = options["circular-save-stats"].GetValue().(bool)
	n.Logln(glightning.Debug, "save stats: ", n.saveStats)

	n.lightning.SetTimeout(DEFAULT_RPC_TIMEOUT)
}

func (n *Node) Logf(level glightning.LogLevel, format string, v ...any) {
	n.plugin.Log(util.GetCallInfo()+fmt.Sprintf(format, v...), level)
}

func (n *Node) Logln(level glightning.LogLevel, v ...any) {
	n.plugin.Log(util.GetCallInfo()+fmt.Sprint(v...), level)
}

func (n *Node) RefreshChannel(channel *graph.Channel) {
	// this is needed to get up-to-date fees and channel info such as state
	channels, err := n.lightning.GetChannel(channel.ShortChannelId)
	if err != nil {
		n.Logln(glightning.Unusual, err)
		return
	}
	n.Graph.RefreshChannels(channels)
}

func (n *Node) Stopped() bool {
	return n.stopped
}

func (n *Node) SetStopped(stopped bool) {
	n.stopped = stopped
}

func (n *Node) GetPeersLock() *sync.RWMutex {
	return n.PeersLock
}

func (n *Node) GetGraph() *graph.Graph {
	return n.Graph
}

func (n *Node) GetId() string {
	return n.Id
}

func (n *Node) GetPeers() map[string]*glightning.Peer {
	return n.Peers
}

func (n *Node) GetFromDb(key string) ([]byte, error) {
	return n.DB.Get(key)
}
