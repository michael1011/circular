package node

import (
	"circular/graph"
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
	actionGetInfo         = "getinfo"
	actionListPeers       = "listpeers"
	actionGetNode         = "getnode"
	actionRebalanceByScid = "rebalancebyscid"

	// Responses
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
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
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
		Data: resData,
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
				return n.lightning.ListPeers()
			})
			break

		case actionGetNode:
			var nodeReq websocketGetNodeRequest
			err = forwardRequest(ws, actionGetNode, data.Data, &nodeReq, func() (any, error) {
				return n.lightning.GetNode(nodeReq.NodeId)
			})
			break

		case actionRebalanceByScid:
			var rebalanceScid types.RebalanceByScid
			err = forwardRequest(ws, actionRebalanceByScid, data.Data, &rebalanceScid, func() (any, error) {
				go func() {
					var res types.Result
					err := n.lightning.Request(&rebalanceScid, &res)
					if err != nil {
						n.websocketBroadcast(actionRebalanceFailed, websocketResponse{
							Error: err.Error(),
						})
						return
					}

					n.websocketBroadcast(actionRebalanceEnd, res)
				}()

				return nil, nil
			})
			break

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

	http.Handle("/", websocket.Handler(n.handleWebsocket))
	go func() {
		err := http.ListenAndServe(endpoint, nil)
		if err != nil {
			log.Fatalln("error starting WebSocket: " + err.Error())
		}
	}()
}

func (n *Node) websocketBroadcast(action string, msg any) {
	n.activeWebSocketsLock.Lock()
	defer n.activeWebSocketsLock.Unlock()

	data := websocketMessage{
		Action: action,
		Data:   msg,
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
	n.websocketBroadcast(actionRebalanceUpdate, route)
}
