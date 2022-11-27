package node

import (
	"encoding/json"
	"github.com/elementsproject/glightning/glightning"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/net/websocket"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	actionGetInfo   = "getinfo"
	actionListPeers = "listpeers"
	actionGetNode   = "getnode"
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

func (n *Node) handleWebsocket(ws *websocket.Conn) {
	var data websocketMessage

	for {
		err := websocket.JSON.Receive(ws, &data)

		if err != nil {
			if err == io.EOF {
				break
			}

			n.Logln(glightning.Info, "could not read WebSocket message: "+err.Error())
			continue
		}

		msg, _ := json.Marshal(data)
		n.Logln(glightning.Info, "got websocket msg: "+string(msg))

		switch strings.ToLower(data.Action) {
		case actionGetInfo:
			var nodeInfo *glightning.NodeInfo
			nodeInfo, err = n.lightning.GetInfo()

			if err != nil {
				err = websocket.JSON.Send(ws, websocketResponse{
					Error: "could not getinfo: " + err.Error(),
				})
				break
			}

			err = websocket.JSON.Send(ws, websocketResponse{
				Data: nodeInfo,
			})
			break

		case actionListPeers:
			var peers []*glightning.Peer
			peers, err = n.lightning.ListPeers()

			if err != nil {
				err = websocket.JSON.Send(ws, websocketResponse{
					Error: "could not listpeers: " + err.Error(),
				})
				break
			}

			err = websocket.JSON.Send(ws, websocketResponse{
				Data: peers,
			})
			break

		case actionGetNode:
			var nodeReq websocketGetNodeRequest
			err = mapstructure.Decode(data.Data, &nodeReq)
			if err != nil {
				err = websocket.JSON.Send(ws, websocketResponse{
					Error: "could not getnode: " + err.Error(),
				})
				break
			}

			var node *glightning.Node
			node, err = n.lightning.GetNode(nodeReq.NodeId)
			if err != nil {
				err = websocket.JSON.Send(ws, websocketResponse{
					Error: "could not getnode: " + err.Error(),
				})
				break
			}

			err = websocket.JSON.Send(ws, websocketResponse{
				Data: node,
			})
			break

		default:
			err = websocket.JSON.Send(ws, websocketResponse{
				Error: "unknown action",
			})
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
