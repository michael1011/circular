package rebalance

import (
	"circular/node"
	"circular/util"
	"github.com/elementsproject/glightning/jrpc2"
)

type ByScidCommand struct {
	OutScid  string     `json:"outscid"`
	InScid   string     `json:"inscid"`
	Amount   uint64     `json:"amount,omitempty"`
	MaxPPM   uint64     `json:"maxppm,omitempty"`
	Attempts int        `json:"attempts,omitempty"`
	MaxHops  int        `json:"maxhops,omitempty"`
	Node     *node.Node `json:"-"`
}

func (r *ByScidCommand) Name() string {
	return "circular"
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
