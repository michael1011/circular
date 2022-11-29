package node

import (
	"circular/graph"
	"circular/rebalance"
	"circular/types"
	"fmt"
	"github.com/elementsproject/glightning/glightning"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/net/websocket"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	// Requests
	actionPing            = "ping"
	actionGetInfo         = "getinfo"
	actionListPeers       = "listpeers"
	actionGetNode         = "getnode"
	actionRebalanceByScid = "rebalancebyscid"
	actionRebalanceStop   = "stop"
	actionRebalanceResume = "resume"

	// Responses
	actionPong            = "pong"
	actionRebalanceUpdate = "rebalanceupdate"
	actionRebalanceEnd    = "rebalanceend"
	actionRebalanceFailed = "rebalancefailed"
)

type websocketMessage struct {
	Action string `json:"action"`
	Data   any    `json:"data"`
}

type websocketGetNodeRequest struct {
	NodeId string `json:"nodeId"`
}

type websocketResponse struct {
	Action string `json:"action"`
	Data   any    `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
}

type channelInfo struct {
	*types.Peer
	Alias string `json:"alias"`
	Color string `json:"color"`
}

func sendActionFailed(ws *websocket.Conn, action string, err error) error {
	return websocket.JSON.Send(ws, websocketResponse{
		Error: fmt.Sprintf("could not %s: %s", action, err),
	})
}

func forwardRequest(ws *websocket.Conn, action string, data, req any, cb func() (any, error)) error {
	if req != nil {
		err := mapstructure.Decode(data, req)
		if err != nil {
			return sendActionFailed(ws, action, err)
		}
	}

	resData, err := cb()
	if err != nil {
		return sendActionFailed(ws, action, err)
	}

	if resData == nil {
		return nil
	}

	return websocket.JSON.Send(ws, websocketResponse{
		Action: action,
		Data:   resData,
	})
}

func (n *Node) handleWebsocket(ws *websocket.Conn) {
	n.activeWebSocketsLock.Lock()
	n.activeWebSockets = append(n.activeWebSockets, ws)
	n.activeWebSocketsLock.Unlock()

	var data websocketMessage

	for {
		err := websocket.JSON.Receive(ws, &data)

		if err != nil {
			if err == io.EOF {
				n.activeWebSocketsLock.Lock()
				for i, wsComp := range n.activeWebSockets {
					if ws == wsComp {
						n.activeWebSockets[i] = n.activeWebSockets[len(n.activeWebSockets)-1]
						n.activeWebSockets = n.activeWebSockets[:len(n.activeWebSockets)-1]
						break
					}
				}
				n.activeWebSocketsLock.Unlock()
				break
			}

			n.Logln(glightning.Info, "could not read WebSocket message: "+err.Error())
			continue
		}

		switch strings.ToLower(data.Action) {
		case actionGetInfo:
			err = forwardRequest(ws, actionGetInfo, data.Data, nil, func() (any, error) {
				return n.lightning.GetInfo()
			})
			break

		case actionListPeers:
			err = forwardRequest(ws, actionListPeers, data.Data, nil, func() (any, error) {
				var res struct {
					Peers []*types.Peer `json:"peers"`
				}

				err := n.lightning.Request(glightning.ListPeersRequest{}, &res)
				if err != nil {
					return nil, err
				}

				channels := make([]*channelInfo, len(res.Peers))
				for i, peer := range res.Peers {
					node, err := n.lightning.GetNode(peer.Id)
					if err != nil {
						return nil, err
					}

					channels[i] = &channelInfo{
						Peer:  peer,
						Alias: node.Alias,
						Color: node.Color,
					}
				}

				return channels, nil
			})
			break

		case actionGetNode:
			var nodeReq websocketGetNodeRequest
			err = forwardRequest(ws, actionGetNode, data.Data, &nodeReq, func() (any, error) {
				return n.lightning.GetNode(nodeReq.NodeId)
			})
			break

		case actionRebalanceByScid:
			var rebalanceScid rebalance.ByScidCommand
			err = forwardRequest(ws, actionRebalanceByScid, data.Data, &rebalanceScid, func() (any, error) {
				go func() {
					var res types.Result
					n.Logln(glightning.Info, "wtf", rebalanceScid)
					err := n.lightning.Request(&rebalanceScid, &res)
					if err != nil {
						n.websocketBroadcast(actionRebalanceFailed, nil, err.Error())
						return
					}

					n.websocketBroadcast(actionRebalanceEnd, res, nil)
				}()

				return nil, nil
			})
			break

		case actionRebalanceResume:
			var res Resume
			err = forwardRequest(ws, actionRebalanceResume, data.Data, &res, func() (any, error) {
				err := n.lightning.Request(&Resume{}, &res)
				if err != nil {
					return nil, err
				}
				return res, nil
			})
			break

		case actionRebalanceStop:
			var res Stop
			err = forwardRequest(ws, actionRebalanceStop, data.Data, &res, func() (any, error) {
				err := n.lightning.Request(&Stop{}, &res)
				if err != nil {
					return nil, err
				}
				return res, nil
			})
			break

		case actionPing:
			err = websocket.JSON.Send(ws, websocketResponse{
				Action: actionPong,
			})

		default:
			err = websocket.JSON.Send(ws, websocketResponse{
				Error: "unknown action",
			})
			break
		}

		if err != nil {
			n.Logln(glightning.Info, "could not write WebSocket message: "+err.Error())
			continue
		}
	}
}

func (n *Node) startWebsocket(options map[string]glightning.Option) {
	endpoint := options[OptionWebSocketEndpoint].GetValue().(string)

	if endpoint == "" {
		n.Logln(glightning.Info, "not enabling WebSocket; endpoint was set to empty string")
		return
	}

	n.Logln(glightning.Info, "enabling WebSocket on: "+endpoint)

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		s := websocket.Server{Handler: websocket.Handler(n.handleWebsocket)}
		s.ServeHTTP(writer, request)
	})
	go func() {
		err := http.ListenAndServe(endpoint, nil)
		if err != nil {
			log.Fatalln("error starting WebSocket: " + err.Error())
		}
	}()
}

func (n *Node) websocketBroadcast(action string, msg any, err any) {
	n.activeWebSocketsLock.Lock()
	defer n.activeWebSocketsLock.Unlock()

	data := websocketResponse{
		Action: action,
		Data:   msg,
	}

	if err != nil {
		data.Error = err.(string)
	}

	for _, ws := range n.activeWebSockets {
		err := websocket.JSON.Send(ws, data)
		if err != nil {
			n.Logln(glightning.Info, "could not broadcast WebSocket message: "+err.Error())
			break
		}
	}
}

func (n *Node) SendRebalanceAttempt(route *graph.PrettyRoute) {
	n.websocketBroadcast(actionRebalanceUpdate, route, nil)
}
