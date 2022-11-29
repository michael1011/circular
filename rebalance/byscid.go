package rebalance

import (
	"circular/singleton"
	"circular/util"
	"github.com/elementsproject/glightning/jrpc2"
)

type ByScidCommand struct {
	OutScid  string `json:"outscid"`
	InScid   string `json:"inscid"`
	Amount   uint64 `json:"amount,omitempty"`
	MaxPPM   uint64 `json:"maxppm,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
	MaxHops  int    `json:"maxhops,omitempty"`
}

func (r *ByScidCommand) Name() string {
	return "circular"
}

func (r *ByScidCommand) New() interface{} {
	return &ByScidCommand{}
}

func (r *ByScidCommand) Call() (jrpc2.Result, error) {
	node := singleton.GetNode()
	if r.InScid == "" || r.OutScid == "" {
		return nil, util.ErrNoRequiredParameter
	}

	outgoingChannel, err := node.GetOutgoingChannelFromScid(r.OutScid)
	if err != nil {
		return nil, err
	}

	incomingChannel, err := node.GetIncomingChannelFromScid(r.InScid)
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
