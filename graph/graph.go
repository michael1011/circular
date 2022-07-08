package graph

import (
	"container/heap"
	"encoding/json"
	"errors"
	"github.com/elementsproject/glightning/glightning"
	"log"
	"os"
)

const (
	GRAPH_REFRESH = "10m"
	FILE          = "graph.json"
)

// Edge contains all the channels going from nodeA to nodeB
// Key is the short channel id
// Value is the channel
type Edge map[string]*Channel

// Graph is the lightning network graph from the perspective of self
// It has been built from the gossip received by lightningd.
// If you want to access the channels flowing out from a node,
// you can use the following: g.Outbound[node]
// If you want to access the channels between nodeA and nodeB,
// you can use the following: g.Outbound[nodeA][nodeB]
// If you want to access a specific channel between nodeA and nodeB,
// you can use the following: g.Outbound[nodeA][nodeB][shortChannelId]
type Graph struct {
	Outbound map[string]map[string]Edge `json:"outbound"`
	Inbound  map[string]map[string]Edge `json:"inbound"`
}

func NewGraph() *Graph {
	g, err := loadFromFile()
	if err != nil {
		g = &Graph{}
	}
	return g
}

func loadFromFile() (*Graph, error) {
	file, err := os.Open(FILE)
	if err != nil {
		file, err = os.Open(FILE + ".old")
		if err != nil {
			return nil, err
		}
	}
	defer file.Close()
	var g Graph
	err = json.NewDecoder(file).Decode(&g)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (g *Graph) SaveToFile() {
	// open temporary file
	file, err := os.Create(FILE + ".tmp")
	if err != nil {
		log.Printf("error opening file: %v", err)
		return
	}
	defer file.Close()
	// write json
	bytes, err := json.Marshal(g)
	_, err = file.Write(bytes)
	if err != nil {
		log.Printf("error writing graph on file: %v", err)
		return
	}

	// save old file
	// check if FILE exists
	if _, err := os.Stat(FILE); err == nil {
		err = os.Rename(FILE, FILE+".old")
	}
	// rename tmp to FILE
	err = os.Rename(FILE+".tmp", FILE)
}

func allocate(links *map[string]map[string]Edge, from, to string) {
	if *links == nil {
		*links = make(map[string]map[string]Edge)
	}
	if (*links)[from] == nil {
		(*links)[from] = make(map[string]Edge)
	}
	if (*links)[from][to] == nil {
		(*links)[from][to] = make(Edge)
	}
}

func (g *Graph) AddChannel(c *glightning.Channel) {
	allocate(&g.Outbound, c.Source, c.Destination)
	allocate(&g.Inbound, c.Destination, c.Source)
	liquidity := estimateInitialLiquidity(c)
	(g.Outbound)[c.Source][c.Destination][c.ShortChannelId] =
		&Channel{*c, liquidity}
	(g.Inbound)[c.Destination][c.Source][c.ShortChannelId] =
		&Channel{*c, (c.Satoshis * 1000) - liquidity}
}

func estimateInitialLiquidity(c *glightning.Channel) uint64 {
	return uint64(0.5 * float64(c.Satoshis*1000))
}

func (g *Graph) GetRoute(src, dst string, amount uint64, exclude map[string]bool) (*Route, error) {
	hops, err := g.dijkstra(src, dst, amount, exclude)
	if err != nil {
		return nil, err
	}
	route := NewRoute(src, dst, amount, hops)
	return route, nil
}

func (g *Graph) dijkstra(src, dst string, amount uint64, exclude map[string]bool) ([]RouteHop, error) {
	// start from the destination and find the source so that we can compute fees
	// TODO: consider that 32bits fees can be a problem but the api does it in that way
	distance := make(map[string]int)
	hop := make(map[string]RouteHop)
	maxDistance := 1 << 31
	for u := range g.Inbound {
		distance[u] = maxDistance
	}
	distance[dst] = 0

	pq := make(PriorityQueue, 1, 16)
	// Insert destination
	pq[0] = &Item{value: &PqItem{
		Node:   dst,
		Amount: amount,
		Delay:  0,
	}, priority: 0}
	heap.Init(&pq)

	for pq.Len() > 0 {
		pqItem := heap.Pop(&pq).(*Item)
		u := pqItem.value.Node
		amount := pqItem.value.Amount
		delay := pqItem.value.Delay
		fee := pqItem.priority
		if u == src {
			break
		}
		if fee > distance[u] {
			continue
		}
		for v, edge := range g.Inbound[u] {
			if exclude[v] {
				continue
			}
			for _, channel := range edge {
				if !channel.canUse(amount) {
					continue
				}
				channelFee := int(channel.computeFee(amount))
				newDistance := distance[u] + channelFee
				if newDistance < distance[v] {
					distance[v] = newDistance
					hop[v] = RouteHop{
						channel,
						amount,
						delay,
					}
					heap.Push(&pq, &Item{value: &PqItem{
						Node:   v,
						Amount: amount + uint64(channelFee),
						Delay:  delay + channel.Delay,
					}, priority: newDistance})
				}
			}
		}
	}
	if distance[src] == maxDistance {
		return nil, errors.New("no route found")
	}
	// now we have the hop map, we can build the hops
	hops := make([]RouteHop, 0, 10)
	for u := src; u != dst; u = hop[u].Channel.Destination {
		hops = append(hops, hop[u])
	}
	return hops, nil
}
