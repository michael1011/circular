package rebalance

import (
	"circular/node"
	"circular/types"
	"circular/util"
	"github.com/elementsproject/glightning/jrpc2"
)

type ByScidCommand struct {
	types.RebalanceByScid
	Node *node.Node `json:"-"`
}

func (r *ByScidCommand) New() interface{} {
	return &ByScidCommand{}
}

func (r *ByScidCommand) Call() (jrpc2.Result, error) {
	r.Node = node.GetNode()
	if r.InScid == "" || r.OutScid == "" {
		return nil, util.ErrNoRequiredParameter
	}

	outgoingChannel, err := r.Node.GetOutgoingChannelFromScid(r.OutScid)
	if err != nil {
		return nil, err
	}

	incomingChannel, err := r.Node.GetIncomingChannelFromScid(r.InScid)
	if err != nil {
		return nil, err
	}

	rebalance := NewRebalance(outgoingChannel, incomingChannel, r.Amount, r.MaxPPM, r.Attempts, r.MaxHops)

	err = rebalance.Setup()
	if err != nil {
		return nil, err
	}

	return rebalance.Run(), nil
}
